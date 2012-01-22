package seven5

import (
	"bytes"
	"fmt"
	"github.com/seven5/mongrel2"
	"net/http"
	"os"
	"runtime"
	"strings"
)

var shutdownMsg = "shutdown has begun"

//m2ToHttp converts a mongrel2 level request to an http one compatible with net/http
func m2ToHttp(in *mongrel2.HttpRequest) (out *http.Request) {

	//for k,v := range in.Header {
	//	fmt.Printf("------->'%s' -> '%s'\n",k,v)
	//}

	method := in.Header["METHOD"]
	url := in.Header["URI"]
	buffer := bytes.NewBuffer(in.Body)
	out, err := http.NewRequest(method, url, buffer)
	if err != nil {
		panic(err)
	}
	header := make(map[string][]string)
	out.Header = header

	for k, v := range in.Header {
		out.Header.Set(http.CanonicalHeaderKey(k), v)
	}
	return
}

//httpToM2 converts an http level response to a mongrel2 one.  the extra parameters are needed
//because of mongrel2's clustering.
func httpToM2(sid string, cid int, in *http.Response) (out *mongrel2.HttpResponse) {
	out = new(mongrel2.HttpResponse)

	out.ServerId = sid
	out.ClientId = []int{cid}

	out.StatusCode = in.StatusCode
	out.StatusMsg = in.Status
	out.Header = make(map[string]string)

	for k, v := range in.Header {
		out.Header[http.CanonicalHeaderKey(k)] = v[0] //should we do this?
	}

	out.ContentLength = in.ContentLength
	out.Body = in.Body

	return
}

//httpRunner is an interface that is used by the infrastructure to indicate that the object can
//"pump" http through it. 
type httpRunner interface {
	mongrel2.HttpHandler
	//runHttp is called by the infrastructure to start the flow of the HTTP requests and responses.
	//The second parameter is where these HTTP requests/responses are processed.  The "target" does
	//not know about or deal with the particulars of the way the requests and response are
	//gathered or squirted back to the client (it doesn't deal with any communications).
	runHttp(config *projectConfig, target Httpified)
}

//httpRunnerDefault is a default implementation of the httpRunner interface for a mongrel2 based
//service.  It uses two Go channels: one for reading requests, the other for writing responses. It
//does not know how to deal with sockets, just channels.  The socket level is handled by the mongrel2
//package.
type httpRunnerDefault struct {
	*mongrel2.HttpHandlerDefault
	In  chan *mongrel2.HttpRequest
	Out chan *mongrel2.HttpResponse
}

//newHttpRunnerDefault returns an httpRunnerDefault that is connected to mongrel2.  This
//centralizes the connection to m2 so we can have more options later.
func newHttpRunnerDefault() *httpRunnerDefault {
	return &httpRunnerDefault{mongrel2.HttpHandlerDefault: &mongrel2.HttpHandlerDefault{new(mongrel2.RawHandlerDefault)}}
}

//RunHttp launches the processing of the HTTP protocol via goroutines.  This call will 
//never return.  It will repeatedly call ProcessRequest() as messages arrive and need
//to be sent to the mongrel2 server.  It is careful to protect itself from code that
//might call panic() even though this is _not_ advised for implementors; it is preferred
//to have implementations that detect a problem use the HTTP error code set.
func (self *httpRunnerDefault) runHttp(config *projectConfig, target Httpified) {

	i := make(chan *mongrel2.HttpRequest)
	o := make(chan *mongrel2.HttpResponse)
	self.In = i
	self.Out = o

	//concurrency claim: this goroutine (ReadLoop) CANNOT be properly killed from outside because 
	//it is possible to be blocked in a C-level call to read (zmq socket).  So it is 
	//possible that we leave it "stranded" during shutdown and then later expect the closing
	//of the ZMQ context to signal with ETERM that it should die.
	go self.ReadLoop(self.In)
	go self.WriteLoop(self.Out)

	for {
		//block until we get a message from the server
		req, ok := <-self.In

		if !ok {
			config.Logger.Printf("[%s]: close of channel in run HTTP!", target.Name())
			//concurrency claim: this is closed by a call to Shutdown() on this same
			//object. That shutdown, however, occurs on a DIFFERENT goroutine and that
			//OTHER goroutine will have been the one who signalled us by closing
			//self.In.  However, the other go routine cannot safely close the self.Out
			//channel because he has no way to be sure that there is not a processing
			//pass "in play" already (this loop could be in any state when he did
			//the close.)  So, we must do the close before we return.  This has the 
			//effect of killing the goroutine launched above on self.Writeloop().
			close(self.Out)
			return
		} else {
			config.Logger.Printf("[%s]: serving %s", target.Name(), req.Path)
		}

		/*for k,v:=range req.Header {
			fmt.Fprintf(os.Stderr,"--->>> header: '%s'='%s'\n",k,v)
		}*/

		//note: mongrel converts this to lower case!
		testHeader := req.Header[strings.ToLower(route_test_header)]
		if target.Name() == testHeader {
			testResp := new(mongrel2.HttpResponse)
			config.Logger.Printf("[ROUTE TEST] %s : %s\n", target.Name(), req.Path)
			testResp.ClientId = []int{req.ClientId}
			testResp.ServerId = req.ServerId
			testResp.StatusCode = route_test_response_code
			testResp.Header = map[string]string{route_test_header: target.Name()}
			testResp.StatusMsg = "Thanks for testing with seven5"
			self.Out <- testResp
			continue
		}

		resp := protectedProcessRequest(config, req, target)
		if resp == nil {
			//special case for shutting down
			resp = new(mongrel2.HttpResponse)
			resp.StatusCode = 200
			resp.StatusMsg = "shutting down"
			self.Out <- resp
			//NOW do the shutdown
			close(getShutdownChannel())
		} else {
			self.Out <- resp
		}
	}
}

//protectedProcessRequest is present to allow us to trap panic's that occur inside the web 
//application.  The web application should not really ever do this, it should generate 500
//pages instead but a nil pointer dereference or similar is possible.
func protectedProcessRequest(config *projectConfig, req *mongrel2.HttpRequest, target Httpified) (resp *mongrel2.HttpResponse) {
	defer func() {
		if x := recover(); x != nil {
			config.Logger.Printf("[%s]: PANIC! sent error page for %s: %v\n", target.Name(), req.Path,x)
			fmt.Fprintf(os.Stderr, "[%s]: PANIC! sent error page for %s\n", target.Name(), req.Path)
			resp = new(mongrel2.HttpResponse)
			resp.StatusCode = 500
			resp.StatusMsg = "Internal Server Error"
			b := NewBufferCloserFromString(fmt.Sprintf("Panic: %v\n", x))
			resp.ContentLength = b.Len()
			resp.Body = b
			fmt.Fprintf(os.Stderr, "Recover result: %s\n-----------\n", generateStackTrace(fmt.Sprintf("%v", x)))
		}
	}()

	//this is the place where we interact with user level code...entering http/Request&Response
	requestAsHttp := m2ToHttp(req)
	respAsHttp := target.ProcessRequest(requestAsHttp)
	if respAsHttp == nil {
		//this special case allows us to handle the shutdown message and *THEN* die
		resp = nil
	} else {
		resp = httpToM2(req.ServerId, req.ClientId, respAsHttp)
		//leaving http/Request&Response
		config.Logger.Printf("[%s]: responded to %s with %d bytes of content\n", target.Name(), req.Path, resp.ContentLength)
	}
	return
}

//generate500Page returns an error page as a mongrel2.Response.  This includes a call stack of the point
//where the caller called this function.
func generate500Page(err string, request *mongrel2.HttpRequest) *mongrel2.HttpResponse {
	fiveHundred := new(mongrel2.HttpResponse)

	fiveHundred.ServerId = request.ServerId
	fiveHundred.ClientId = []int{request.ClientId}

	fiveHundred.StatusCode = 500
	fiveHundred.StatusMsg = "Internal Server Error"
	b := NewBufferCloserFromString(generateStackTrace(fmt.Sprintf("%v", err)))
	fiveHundred.Body = b
	fiveHundred.ContentLength = b.Len()
	return fiveHundred
}

//generateStackTrace knows how to create a string from an error by using runtime.Caller and filtering
//out a couple of calls, such as itself.
func generateStackTrace(err string) string {
	buffer := new(bytes.Buffer)
	buffer.WriteString(err)
	buffer.WriteString("\n----Stacktrace ----\n")
	for i := 2; ; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok {
			if last := isGoRuntime(file); last != "" {
				file = fmt.Sprintf("[Go Runtime %s]", last)
			}
			s := fmt.Sprintf("%s: line %d\n", file, line)
			buffer.WriteString(s)
		} else {
			break
		}
	}
	return buffer.String()
}

//isGoRuntime looks for the pattern go/src/pkg/runtime in the path to see if this file is likely
//to be one we can ignore.  it returns the last part of the path if finds the pattern otherwise ""
func isGoRuntime(file string) string {
	if strings.Index(file, "/go/src/pkg/runtime/") != -1 {
		split := strings.Split(file, "/")
		return split[len(split)-1]
	}
	return ""
}

//Shutdown here is a bit trickier than it might look.  Concurrency claim:  we close ONLY the
//read channel that is in use by runHttp().  We cannot safely close the write channel
//in use in runHttp() because there may be a request being processed (on runHttp's goroutine)
//when this method gets called.  By closing the read channel only we insure that the other
//goroutine will run until completion of its-in progress request and that will mean using
//the write channel.  (If we closed the write channel also, the other goroutine could get
//an "attempt to write on closed channel" panic.) This means that the other side of this signal
// (runHttp)  is responsible for cleaning up the out channel.
func (self *httpRunnerDefault) Shutdown() {
	//fmt.Printf("shutdown called: about to signal on a read channel\n")
	//check for nil because it may not be initialized yet...
	if self.In != nil {
		close(self.In)
	}
}

//BindToTransport connects this runner to the lower level of the implementation, which for 
//mongrel2
func (self *httpRunnerDefault) BindToTransport(name string, transport Transport) error {
	t := transport.(*zmqTransport)

	if err := mongrel2.RawHandler(self).Bind(name, t.Ctx); err != nil {
		fmt.Fprintf(os.Stderr, "unable to bind %s to socket! %s\n", name, err)
		return err
	}
	return nil
}