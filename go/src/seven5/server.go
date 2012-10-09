package seven5

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//defines a new error type
var BAD_GOPATH = errors.New("GOPATH is not defined or is empty")

//AddIndexAndFind maps the singular and plural names into the url space.  The names should not include 
//any slashes or spaces as this will trigger armageddon and destroy all life on this planet.  If either
//name is "" the corresponding interface value is ignored.  The final interface should be a struct
//(not a pointer to a struct) that describes the json values exchanged over the wire.  The Finder
//and Indexer are expected (but not required) to be marshalling these values as returned objects.
//The Finder and Indexer are called _only_ in response to a GET method on the appropriate URI.
//
//The marshalling done in seven5.JsonResult uses the go json package, so the struct field tags using
//"json" will be respected.  The struct must contain an int32 field called Id.  The url space uses
//lowercase only, so the singular and plural will be converted.  If both singular and plural are
//"" this function computes the capital of North Ossetia and ignores it.
func (self *SimpleHandler) AddFindAndIndex(singular string, finder Finder, plural string,
	indexer Indexer, r interface{}) {

	d := NewResourceDoc(r)

	if singular != "" {
		withSlashes := fmt.Sprintf("/%s/", strings.ToLower(singular))
		self.resource[withSlashes] = finder
		self.mux.Handle(withSlashes, self)
		self.doc[withSlashes] = d
		d.Find = finder
		d.GETSingular = singular
	}
	if plural != "" {
		withSlashes := fmt.Sprintf("/%s/", strings.ToLower(plural))
		self.resource[withSlashes] = indexer
		self.mux.Handle(withSlashes, self)
		self.doc[withSlashes] = d
		d.Index = indexer
		d.GETPlural = plural
	}
}

//ServeHTTP allows this object to act like an http.Handler. ServeHTTP data is passed to Dispatch
//after some minimal processing.  This is not used in tests, only when on a real network.
func (self *SimpleHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	hdr := toSimpleMap(req.Header)
	qparams := toSimpleMap(map[string][]string(req.URL.Query()))
	json, err := self.Dispatch(req.Method, req.URL.Path, hdr, qparams)
	if err!=nil && err.StatusCode==http.StatusNotFound {
		self.mux.ServeHTTP(writer,req)
	} else {
		dumpOutput(writer, json, err)
	}
}

//SimpleHandler is the default implementation of the Handler interface that ignores multiple values for
//headers and query params because these are both rare and error-prone.  All resources need to be
//added to the Handler before it starts serving real HTTP requests.
type SimpleHandler struct {
	//resource maps names in URL space to objects that implement one or more of our rest interfaces
	resource map[string]interface{}
	//connection to the http layer
	mux *http.ServeMux
	//doc handling
	doc map[string]*ResourceDoc
}

//NewSimpleHandler creates a new SimpleHandler with an empty mapping in the URL space. 
func NewSimpleHandler() *SimpleHandler {
	return &SimpleHandler{resource: make(map[string]interface{}),
		mux: http.NewServeMux(),
		doc: make(map[string]*ResourceDoc)}
}
//ServeMux returns the underlying ServeMux that can be used to register additional HTTP
//resources (paths) with this object.
func (self *SimpleHandler) ServeMux() *http.ServeMux {
	return self.mux
}

//Dispatch does the dirty work of finding a resource and calling it.
//It returns the value from the correct rest-level function or an error.
//It generates some errors itself if, for example a 404 or 501 is needed.
//I borrowed lots of ideas and inspiration from "github.com/Kissaki/rest2go"
func (self *SimpleHandler) Dispatch(method string, uriPath string, header map[string]string,
	queryParams map[string]string) (string, *Error) {

	matched, id, someResource := self.resolve(uriPath)
	if matched == "" {
		return NotFound()
	}
	switch method {
	case "GET":
		if len(id) == 0 {
			if resIndex, ok := someResource.(Indexer); ok {
				return resIndex.Index(header, queryParams)
			} else {
				//log.Printf("%T isn't an Indexer, returning NotImplemented", someResource)
				return NotImplemented()
			}
		} else {
			// Find by ID
			var num int64
			var err error
			if num, err = strconv.ParseInt(id, 10, 64); err != nil {
				return BadRequest("resource ids must be non-negative integers")
			}
			//resource id is a number, try to find it
			if resFind, ok := someResource.(Finder); ok {
				return resFind.Find(Id(num), header, queryParams)
			} else {
				return NotImplemented()
			}
		}
	}
	return "", &Error{http.StatusNotImplemented, "", "Not implemented yet"}
}

//resolve is used to find the matching resource for a particular request.  It returns the match
//and the resource matched.  If no match is found it returns "",nil.  resolve does not check
//that the resulting object is suitable for any purpose, only that it matches.
func (self *SimpleHandler) resolve(path string) (string, string, interface{}) {
	someResource, ok := self.resource[path]
	var id string
	result := path

	if !ok {
		// no resource found, thus check if the path is a resource + ID
		i := strings.LastIndex(path, "/")
		if i == -1 {
			return "", "", nil
		}
		// Move index to after slash as that’s where we want to split
		i++
		id = path[i:]
		var uriPathParent string
		uriPathParent = path[:i]
		someResource, ok = self.resource[uriPathParent]
		if !ok {
			return "", "", nil
		}
		result = uriPathParent
	}
	return result, id, someResource

}

//dumpOutput send the output to the calling client over the network.  Not used in tests,
//only when running against real network.
func dumpOutput(response http.ResponseWriter, json string, err *Error) {
	if err != nil && json != "" {
		log.Printf("ignoring json reponse (%d bytes) because also have error func %+v", len(json), err)
	}
	if err != nil {
		if err.Location != "" {
			response.Header().Add("Location", err.Location)
		}
		http.Error(response, err.Message, err.StatusCode)
		return
	}
	if _, err := response.Write([]byte(json)); err != nil {
		log.Printf("error writing json response %s", err)
	}
	return
}

//BadRequest returns an error struct representing a 402, Bad Request HTTP response. This should be returned
//when the parameters passed the by client don't make sense.
func BadRequest(msg string) (string, *Error) {
	return "", &Error{http.StatusBadRequest, "", fmt.Sprintf("BadRequest - %s ", msg)}
}

//NoContent Returns an error struct representing a "succcess" in the sense of the protocol but 
//a semantic error of "empty".
func NoContent() (string, *Error) {
	return "", &Error{http.StatusNoContent, "", "No content"}
}

//NotFound Returns a 'these are not the droids you're looking for... move along.'
func NotFound() (string, *Error) {
	return "", &Error{http.StatusNotFound, "", "Not found"}
}

//NotImplemented returns an http 501.  This happens if we find a resource at the URL _but_ the 
//implementing struct doesn't have the correct type, for example /foobars/ is a known mapping
//but the struct does not implement Indexer.
func NotImplemented() (string, *Error) {
	return "", &Error{http.StatusNotImplemented, "", "Not implemented"}
}

//InternalErr is a convenience for returning a 501 when an error has been found at the go level.
func InternalErr(err error) (string, *Error) {
	return "", &Error{http.StatusInternalServerError, "", err.Error()}
}

//toSimpleMap converts an http level map with multiple strings as value to single string value.
func toSimpleMap(m map[string][]string) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = strings.TrimSpace(v[0])
	}
	return result
}

//JsonResult returns a json string from the supplied value or return an error (caused by the encoder)
//via the InternalErr function.  This is the normal path for functions that return Json values.
//pretty can be set to true for pretty-printed json.
func JsonResult(v interface{}, pretty bool) (string, *Error) {
	var buff []byte
	var err error

	if pretty {
		buff, err = json.MarshalIndent(v, "", " ")
	} else {
		buff, err = json.Marshal(v)
	}
	if err != nil {
		return InternalErr(err)
	}
	result := string(buff)
	return strings.Trim(result, " "), nil
}

//ProjDirectoryFromGOPATH computes a directory inside the project level of a seven5 project
//that has the default layout.  For a project foo
// foo/
//    dart/
//    db/
//    go/
//         bin/
//         pkg/
//         src/
//               foo/
//    web/
// 
func ProjDirectoryFromGOPATH(rootDir string) (string, error) {
	env := os.Getenv("GOPATH")
	if env == "" {
		return "", BAD_GOPATH
	}
	pieces := strings.Split(env, ":")
	if len(pieces) > 1 {
		env = pieces[0]
	}
	return filepath.Join(filepath.Dir(env), rootDir), nil
}

//seven5StaticContent adds an http handler for static content in a subdirectory
func StaticContent(h Handler, urlPath string, subdir string) {
	//setup static content
	truePath, err := ProjDirectoryFromGOPATH(subdir)
	if err != nil {
		log.Fatalf("can't understand GOPATH or not using default project layout: %s", err)
	}
	//strip the path from requests so that /urlPath/fart = modena/subdir/fart
	h.ServeMux().Handle(urlPath, http.StripPrefix(urlPath, http.FileServer(http.Dir(truePath))))
}

//ListenAndServeDefaultLayout adds the resources that we expect to be present for a typical
//seven5 project to the handler provided and returns it as as http.Handler so it can be
//use "in the normal way" with http.ServeHttp
func AddDefaultLayout(h Handler) http.Handler {
	StaticContent(h, "/static/", "static")
  StaticContent(h, "/dart/", "dart")
	return h
}
