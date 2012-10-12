package italy

import (
  // "github.com/seven5/seven5"
	"seven5"
	"fmt"
	"strings"
	"strconv"
)

//sub structure used for a latitude, longitude pair
type LatLng struct {
	Latitude seven5.Floating
	Longitude seven5.Floating
}

//rest resource for a single city, properties must be public for JSON encoder
type ItalianCity struct {
	Id seven5.Id
	Name seven5.String255
	Population seven5.Integer
	Province seven5.String255
	Location *LatLng
}

//sample data to work with... so no need for DB
var cityData = []*ItalianCity{
	&ItalianCity{Id:0,Name:"Turin", Province:"Piedmont", Population:900569, 
		Location:&LatLng{45.066667, 7.7} },
	&ItalianCity{1,"Milan", 3083955, "Lombardy", &LatLng{45.464167, 9.190278}},
	&ItalianCity{2,"Genoa",800709,"Liguria", &LatLng{44.411111, 8.932778}},
}


//rest resource for the city list, no data used internally because it is stateless
type ItalianCitiesResource struct{
}
//rest resource for a particular city, stateless
type ItalianCityResource struct{
}

//Index returns a list of italian cities, filtered by the prefix header and the maximum
//number returned controlled by the max parameter.  
func (STATELESS *ItalianCitiesResource) Index(headers map[string]string, 
	qp map[string]string) (string,*seven5.Error) {
		
	result := []*ItalianCity{}
	prefix, hasPrefix := headers["Prefix"] //note the capital is always there on headers
	maxStr, hasMax := qp["max"]
	var max int
	var err error
	
	if hasMax {
		if max, err = strconv.Atoi(maxStr); err!=nil {
			return seven5.BadRequest(fmt.Sprintf("can't undestand max parameter %s",maxStr))
		}
	}
	for _, v := range cityData {
		if hasPrefix && !strings.HasPrefix(string(v.Name), prefix) {
				continue
		}
		result = append(result, v)
		if hasMax && len(result)==max {
			break
		}
	}
	return seven5.JsonResult(result,true)
}

//used to create dynamic documentation/api
func (STATELESS *ItalianCitiesResource) IndexDoc() []string {
	return []string{""+
	"The resource `/italiancities/` returns a list of known cities in Italy.  Each element of the list is" +
	"a resource of type italiancity that can be fetched individually at `/italiancity/id`.",
	"italiancities ignores the headers supplied in the GET request.",
	"italiancities ignores the query parameters supplied in the URL to GET.",
	
	"The resource /italiancities/ understands the header 'prefix' and if this header is supplied "+
	"only cities whose Name field begins with the prefix given will be returned.",
	
	"The resource /italiancities/ allows a query parameter 'max' to control the maximum number "+
	"of cities returned.  No guarantee is made about the order of the returned items. Max must "+
	"be a positive integer (not zero).",
	}
}

//given an id, find the object it referencs and return JSON for it. This ignores
//the query parameters but understands the header 'Round' for rounding pop figures to
//100K boundaries.
func (STATELESS *ItalianCityResource) Find(id seven5.Id, hdrs map[string]string, 
	query map[string]string) (string,*seven5.Error) {
	
	r, hasRound := hdrs["Round"] //note the capital is always there on headers
	n := int64(id)
	if n<0 || n>=int64(len(cityData)) {
		return seven5.BadRequest(fmt.Sprintf("id must be from 0 to %d",len(cityData)-1))
	}
	pop:= cityData[id].Population
	if hasRound && strings.ToLower(r)=="true" {
		excess := cityData[id].Population % 100000
		pop -= excess;
		if excess>=50000 {
			pop+=100000
		}
	} 
	data := cityData[id]
	forClient := &ItalianCity{data.Id, data.Name, pop, data.Province, data.Location}
	return seven5.JsonResult(forClient,true)
}

//used to generate documentation/api
func (STATELESS *ItalianCityResource) FindDoc() []string {
	return []string{""+
	"A resource representing a specific italian city at `/italiancity/123`.",
	"The header 'Round' can be used to get population values rounded to the nearest 100K."+
	"Legal values are true, false, and omitted (which means false).",
	"Ignores query parameters.",
	}
}

