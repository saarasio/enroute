package webhttp

import (
	"bytes"
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/saarasio/enroute/enroute-dp/saaras"
	"github.com/saarasio/enroute/enroute-dp/saarasconfig"
	"net/http"

	"github.com/sirupsen/logrus"
)

type FilterConfig struct {
	Filter_name   string `json:"filter_name" xml:"filter_name" form:"filter_name" query:"filter_name"`
	Filter_type   string `json:"filter_type" xml:"filter_type" form:"filter_type" query:"filter_type"`
	Filter_config string `json:"filter_config" xml:"filter_config" form:"filter_config" query:"filter_config"`
}

// @Summary Get all filters in db
// @Description Get all filters in db
// @Tags filter
// @Accept  json
// @Produce  json
// @Success 200 {} int OK
// @Router /filter [get]
// @Security ApiKeyAuth
func GET_FilterConfig(c echo.Context) error {
	var buf bytes.Buffer

	var args map[string]string
	args = make(map[string]string)

	Q := `

query get_all_filterconfig {
    saaras_db_filter {
    filter_id
    filter_name
    filter_type
    config_json
  }
}
    `
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQuery(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	buf2 := bytes.Replace(buf.Bytes(), []byte("config_json"), []byte("filter_config"), -1)

	return c.JSONBlob(http.StatusOK, buf2)
}

// @Summary Get filters detail for provided name
// @Description Get filters detail for provided name
// @Tags filter
// @Param filter_name path string true "Name of filter_name for which to list config"
// @Accept  json
// @Produce  json
// @Success 200 {} int OK
// @Router /filter/{filter_name} [get]
// @Security ApiKeyAuth
func GET_One_FilterConfig(c echo.Context) error {
	var buf bytes.Buffer

	var args map[string]string
	args = make(map[string]string)

	Q := `

query get_all_filter($filter_name: String!) {
  saaras_db_filter(where: {filter_name: {_eq: $filter_name}}) {
    filter_id
    filter_name
    filter_type
    config_json
  }
}

`
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	filter_name := c.Param("filter_name")
	args["filter_name"] = filter_name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQuery(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	buf2 := bytes.Replace(buf.Bytes(), []byte("config_json"), []byte("filter_config"), -1)

	return c.JSONBlob(http.StatusOK, buf2)
}

func setConfigJson(log *logrus.Entry, filter_type string, filter_config string, args *map[string]interface{}) {
	switch filter_type {
	case saarasconfig.FILTER_TYPE_RT_RATELIMIT:
		cfg, err := saarasconfig.UnmarshalRateLimitRouteFilterConfig(filter_config)
		if err == nil {
			(*args)["config_json"] = cfg
		} else {
			(*args)["config_json"] = struct {
				Descriptors [1]string `json:"descriptors"`
			}{
				Descriptors: [...]string{"{}"},
			}
			log.Errorf("Failed to decode [%+v] \n", filter_config)
		}
	case saarasconfig.FILTER_TYPE_HTTP_LUA:
		// Lua filter config contains lua script, which cannot be converted to json
		// convert it to { "config" : "..." } type
		type luaFilterConfig struct {
			Config string `json:"config"`
		}

		var lfc luaFilterConfig
		lfc.Config = filter_config
		log.Errorf("Setting config_json to [%+v] \n", lfc)
		(*args)["config_json"] = lfc
	default:
		// Unsupported filter
		log.Errorf("Unsupported filter type [%s]\n", filter_type)
	}
}

func isFilterTypeValid(filter_type string) bool {
	switch filter_type {
	case saarasconfig.FILTER_TYPE_HTTP_LUA:
		return true
	case saarasconfig.FILTER_TYPE_RT_RATELIMIT:
		return true
	default:
		return false
	}
	return false
}

// @Summary Create filter
// @Description Create filter
// @Tags filter
// @Accept  json
// @Produce  json
// @Param Name body webhttp.FilterConfig true "Name of filter to create"
// @Success 200 {} int OK
// @Router /filter [post]
// @Security ApiKeyAuth
func POST_FilterConfig(c echo.Context) error {
	var buf bytes.Buffer

	var args map[string]interface{}
	args = make(map[string]interface{})

	Q := `

mutation insert_into_filter($filter_name: String!, $filter_type: String, $filter_config: String!, $config_json: jsonb) {
  insert_saaras_db_filter
  (
    objects:
    {
        filter_name: $filter_name,
        filter_type: $filter_type,
        filter_config: $filter_config,
        config_json: $config_json
      },
    on_conflict: { constraint:filter_pkey, update_columns: update_ts }
  ) 
  {
    affected_rows
  }
}
`
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	fc := new(FilterConfig)
	if err := c.Bind(fc); err != nil {
		return err
	}

	if len(fc.Filter_name) == 0 {
		return c.JSON(http.StatusBadRequest, "{\"Error\" : \" Please provide name of Filter using Filter_name field \"}")
	}

	args["filter_name"] = fc.Filter_name
	args["filter_type"] = fc.Filter_type
	args["filter_config"] = fc.Filter_config

	if !isFilterTypeValid(fc.Filter_type) {
		return c.JSON(http.StatusBadRequest, "{\"Error\" : \"Invalid filter type \"}")
	}

	setConfigJson(log, fc.Filter_type, fc.Filter_config, &args)

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQueryGenericVals(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

func getFilterType(filter_name string) string {

	var Q = `
	query GetFilterTypeForFilterName($filter_name: String!) {
		saaras_db_filter(where: {filter_name: {_eq: $filter_name}}) {
			filter_type
		}
	}
	`
	var buf bytes.Buffer

	var args map[string]interface{}
	args = make(map[string]interface{})
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	args["filter_name"] = filter_name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQueryGenericVals(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	type Q_Response struct {
		Data struct {
			SaarasDbFilter []struct {
				FilterType string `json:"filter_type"`
			} `json:"saaras_db_filter"`
		} `json:"data"`
	}

	var qr Q_Response
	err := json.Unmarshal(buf.Bytes(), &qr)

	if err != nil {
		return ""
	}

	var filter_type string

	for _, onef := range qr.Data.SaarasDbFilter {
		filter_type = onef.FilterType
	}

	return filter_type
}

// @Summary Set filter config from file
// @Description Set filter config from file
// @Tags filter
// @Param filter_name path string true "Name of filter_name for which to list config"
// @Accept  json
// @Produce  json
// @Success 200 {} int OK
// @Router /filter/:filter_name/config [post]
// @Security ApiKeyAuth
func POST_One_FilterConfig(c echo.Context) error {
	var buf bytes.Buffer

	var args map[string]interface{}
	args = make(map[string]interface{})

	Q := `

    mutation update_filter($filter_name: String!, $filter_config: String!, $config_json: jsonb) {
        update_saaras_db_filter (
            where:
            {
                filter_name: {_eq: $filter_name}
            }
            _set:
            {
                filter_config: $filter_config
                config_json: $config_json
            }
        )
        {
            affected_rows
        }
    }
`
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	filter_name := c.Param("filter_name")

	// Read config from file
	file, err := c.FormFile("Config")
	if file == nil {
		return c.JSON(http.StatusBadRequest, "{\"Error\" : \"Config empty\"}")
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	buf2 := new(bytes.Buffer)
	buf2.ReadFrom(src)
	config_from_file := buf2.String()

	// We already have the globalconfig_name from params
	// gc_name := c.Param("globalconfig_name")

	if len(filter_name) == 0 {
		return c.JSON(http.StatusBadRequest, "{\"Error\" : \"Please provide filter name\"}")
	}

	args["filter_name"] = filter_name
	args["filter_config"] = config_from_file

	// TODO: Check filter_type is one of the allowed types
	filter_type := getFilterType(filter_name)

	if !isFilterTypeValid(filter_type) {
		return c.JSON(http.StatusBadRequest, "{\"Error\" : \"Cannot find filter or type not recognized\n\"}")
	}

	setConfigJson(log, filter_type, config_from_file, &args)

	log.Errorf("config_json set to [%s]\n", args["config_json"])

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQueryGenericVals(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

// @Summary Update filter config - update filter type
// @Description Update filter config - update filter type
// @Tags filter
// @Param filter_name path string true "Name of filter_name for which to list config"
// @Accept  json
// @Produce  json
// @Param Name body webhttp.FilterConfig true "Name of filter to create"
// @Success 200 {} int OK
// @Router /filter/:filter_name [patch]
// @Security ApiKeyAuth
func PATCH_One_FilterConfig(c echo.Context) error {
	var buf bytes.Buffer

	var args map[string]interface{}
	args = make(map[string]interface{})

	Q := `
	mutation update_filter($filter_name: String!, $filter_type: String!) {
		update_saaras_db_filter (
			where:
			{
				filter_name: {_eq: $filter_name}
			}
			_set:
			{
				filter_type: $filter_type                
			}
		)
		{
			affected_rows
		}
	}
`
	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	filter_name := c.Param("filter_name")

	fc := new(FilterConfig)
	if err := c.Bind(fc); err != nil {
		return err
	}

	if len(fc.Filter_type) == 0 {
		return c.JSON(http.StatusBadRequest, "{\"Error\" : \" Please provide name of Filter using Filter_name field \"}")
	}

	if !isFilterTypeValid(fc.Filter_type) {
		return c.JSON(http.StatusBadRequest, "{\"Error\" : \"Cannot find filter or type not recognized\n\"}")
	}

	args["filter_name"] = filter_name
	args["filter_type"] = fc.Filter_type

	log.Errorf("config_json set to [%s]\n", args["config_json"])

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQueryGenericVals(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

// @Summary Delete filter config
// @Description Delete filter config
// @Tags filter
// @Param filter_name path string true "Name of filter_name for which to list config"
// @Accept  json
// @Produce  json
// @Success 200 {} int OK
// @Router /filter/{filter_name} [delete]
// @Security ApiKeyAuth
func DELETE_FilterConfig(c echo.Context) error {
	var buf bytes.Buffer

	var args map[string]string
	args = make(map[string]string)

	Q := `

mutation delete_filter($filter_name: String!) {
  delete_saaras_db_filter (
    where:
    {
      filter_name: {_eq: $filter_name}
    }
  )
  {
    affected_rows
  }
}

`

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	filter_name := c.Param("filter_name")
	args["filter_name"] = filter_name

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"
	if err := saaras.RunDBQuery(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}

	return c.JSONBlob(http.StatusOK, buf.Bytes())
}

// @Summary Associate a filter with route
// @Description Associate a filter with route
// @Tags filter route
// @Param service_name path string true "Name of service"
// @Param route_name path string true "Name of route"
// @Param filter_name path string true "Name of filter"
// @Accept  json
// @Produce  json
// @Success 200 {} int OK
// @Router /service/{service_name}/route/{route_name}/filter/{filter_name} [post]
// @Security ApiKeyAuth
func POST_Service_Route_FilterConfig_Association(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	route_name := c.Param("route_name")
	filter_name := c.Param("filter_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["service_name"] = service_name
	args["route_name"] = route_name
	args["filter_name"] = filter_name

	var Q = `

mutation insert_into_route_filter($service_name : String!, $route_name : String!, $filter_name : String!) {
    insert_saaras_db_route_filter(
      objects: 
      {
        filter: {
          data: 
          {
            filter_name: $filter_name
          } on_conflict: {constraint: filter_filter_name_key, update_columns: update_ts}
        }, 
        route: {
          data: 
          {
            route_name: $route_name, 
            service: {
              data: 
              {
                service_name: $service_name
              } on_conflict: {constraint: service_service_name_key, update_columns: update_ts}
            }
          }, 
          on_conflict: {constraint: route_service_id_route_name_key, update_columns: update_ts}}
      } on_conflict: {constraint: route_filter_pkey, update_columns: update_ts}
    ) {
    affected_rows
  }
}

`

	if err := saaras.RunDBQuery(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

// @Summary Associate a filter with service
// @Description Associate a filter with service
// @Tags filter service
// @Param service_name path string true "Name of service"
// @Param filter_name path string true "Name of filter"
// @Accept  json
// @Produce  json
// @Success 200 {} int OK
// @Router /service/{service_name}/filter/{filter_name} [post]
// @Security ApiKeyAuth
func POST_Service_FilterConfig_Association(c echo.Context) error {
	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	filter_name := c.Param("filter_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["service_name"] = service_name
	args["filter_name"] = filter_name

	var Q = `

mutation insert_into_service_filter($service_name : String!, $filter_name : String!) {
    insert_saaras_db_service_filter(
      objects:
      {
        filter: {
          data:
          {
            filter_name: $filter_name
          } on_conflict: {constraint: filter_filter_name_key, update_columns: update_ts}
        },
        service: {
          data:
          {
            service_name: $service_name,
          },
          on_conflict: {constraint:service_service_name_key, update_columns: update_ts}}
      } on_conflict: {constraint:service_filter_pkey, update_columns: update_ts}
    ) {
    affected_rows
  }
  
}

`
	if err := saaras.RunDBQuery(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

// @Summary Return filter associated with a service
// @Description Return filter associated with a service
// @Tags filter
// @Param service_name path string true "Name of service"
// @Param filter_name path string true "Name of filter"
// @Accept  json
// @Produce  json
// @Success 200 {} int OK
// @Router /service/{service_name}/filter/{filter_name} [get]
// @Security ApiKeyAuth
func GET_Service_FilterConfig_Association(c echo.Context) error {

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	filter_name := c.Param("filter_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["service_name"] = service_name
	args["filter_name"] = filter_name

	var Q = `

query get_filter_for_service($service_name: String!, $filter_name:String!) {
  saaras_db_filter(
    where :
    {
      _and:[
        {filter_name: {_eq: $filter_name}},
        {service_filters: {service: {service_name: {_eq: $service_name}}}}
      ]
    }
  ) {
    filter_id
    filter_name
    filter_type
    config_json
  }
}
`
	if err := saaras.RunDBQuery(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

// @Summary Return filters associated with a service
// @Description Return filters associated with a service
// @Tags filter
// @Param service_name path string true "Name of service"
// @Accept  json
// @Produce  json
// @Success 200 {} int OK
// @Router /service/{service_name}/filter [get]
// @Security ApiKeyAuth
func GET_Service_All_FilterConfig_Association(c echo.Context) error {

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["service_name"] = service_name

	var Q = `

query get_filter_for_service($service_name: String!) {
  saaras_db_filter(
    where :
    {
      _and:[
        {service_filters: {service: {service_name: {_eq: $service_name}}}}
      ]
    }
  ) {
    filter_id
    filter_name
    filter_type
    config_json
  }
}

`

	if err := saaras.RunDBQuery(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

// @Summary Delete filter associated with a service
// @Description Delete filter associated with a service
// @Tags filter
// @Param service_name path string true "Name of service"
// @Param filter_name path string true "Name of filter for route"
// @Accept  json
// @Produce  json
// @Success 200 {} int OK
// @Router /service/{service_name}/filter/{filter_name} [delete]
// @Security ApiKeyAuth
func DELETE_Service_FilterConfig_Association(c echo.Context) error {

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	filter_name := c.Param("filter_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["service_name"] = service_name
	args["filter_name"] = filter_name

	var Q = `

mutation delete_service_filter($service_name: String!, $filter_name:String!) {
  delete_saaras_db_service_filter
  (
    where : 
    {
      _and:[
        {filter: {filter_name: {_eq: $filter_name}}},
        {service: {service_name: {_eq: $service_name}}}
      ] 
    }
  ) {
   affected_rows
  } 
}

`

	if err := saaras.RunDBQuery(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

// @Summary Return filter associated with a service route
// @Description Return filter associated with a service route
// @Tags filter
// @Param service_name path string true "Name of service for route"
// @Param route_name path string true "Name of route for service"
// @Param filter_name path string true "Name of filter for route"
// @Accept  json
// @Produce  json
// @Success 200 {} int OK
// @Router /service/{service_name}/route/{route_name}/filter/{filter_name} [get]
// @Security ApiKeyAuth
func GET_Service_Route_FilterConfig_Association(c echo.Context) error {

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	route_name := c.Param("route_name")
	filter_name := c.Param("filter_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["service_name"] = service_name
	args["route_name"] = route_name
	args["filter_name"] = filter_name

	var Q = `

query get_filter_for_service_route2($service_name: String!, $route_name: String!, $filter_name:String!) {
  saaras_db_filter(
    where : 
    {
      _and:[
        {filter_name: {_eq: $filter_name}},
        {route_filters: {route: {service: {service_name: {_eq: $service_name}}}}},
        {route_filters: {route: {route_name: {_eq: $route_name}}}}
      ] 
    }
  ) {
    filter_id
    filter_name
    filter_type
    config_json
  } 
}

`

	if err := saaras.RunDBQuery(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

// @Summary Return filters associated with a service route
// @Description Return filters associated with a service route
// @Tags filter
// @Param service_name path string true "Name of service for route"
// @Param route_name path string true "Name of route for service"
// @Accept  json
// @Produce  json
// @Success 200 {} int OK
// @Router /service/{service_name}/route/{route_name}/filter [get]
// @Security ApiKeyAuth
func GET_Service_Route_All_FilterConfig_Association(c echo.Context) error {

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	route_name := c.Param("route_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["service_name"] = service_name
	args["route_name"] = route_name

	var Q = `

query get_filter_for_service_route2($service_name: String!, $route_name: String!) {
  saaras_db_filter(
    where : 
    {
      _and:[
        {route_filters: {route: {service: {service_name: {_eq: $service_name}}}}},
        {route_filters: {route: {route_name: {_eq: $route_name}}}}
      ] 
    }
  ) {
    filter_id
    filter_name
    filter_type
    config_json
  } 
}

`

	if err := saaras.RunDBQuery(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

// @Summary Delete filter associated with a service route
// @Description Delete filter associated with a service route
// @Tags filter
// @Param service_name path string true "Name of service"
// @Param route_name path string true "Name of route for service"
// @Param filter_name path string true "Name of filter for route"
// @Accept  json
// @Produce  json
// @Success 200 {} int OK
// @Router /service/{service_name}/route/{route_name}/filter/{filter_name} [delete]
// @Security ApiKeyAuth
func DELETE_Service_Route_FilterConfig_Association(c echo.Context) error {

	var buf bytes.Buffer
	var args map[string]string
	args = make(map[string]string)

	log2 := logrus.StandardLogger()
	log := log2.WithField("context", "web-http")

	service_name := c.Param("service_name")
	route_name := c.Param("route_name")
	filter_name := c.Param("filter_name")

	url := "http://" + HOST + ":" + PORT + "/v1/graphql"

	args["service_name"] = service_name
	args["route_name"] = route_name
	args["filter_name"] = filter_name

	var Q = `

mutation delete_route_filter($service_name: String!, $route_name: String!, $filter_name:String!) {
  delete_saaras_db_route_filter
  (
    where : 
    {
      _and:[
        {filter: {filter_name: {_eq: $filter_name}}},
        {route: {service: {service_name: {_eq: $service_name}}}},
        {route: {route_name: {_eq: $route_name}}}
      ] 
    }
  ) {
   affected_rows
  } 
}

`

	if err := saaras.RunDBQuery(url, Q, &buf, args, log); err != nil {
		log.Errorf("Error when running http request [%v]\n", err)
	}
	return c.JSONBlob(http.StatusCreated, buf.Bytes())
}

func Add_filter_routes(e *echo.Echo) {
	// Proxy config CRUD
	e.GET("/filter", GET_FilterConfig)
	e.GET("/filter/:filter_name", GET_One_FilterConfig)
	e.POST("/filter", POST_FilterConfig)
	e.POST("/filter/:filter_name/config", POST_One_FilterConfig)
	e.PATCH("/filter/:filter_name", PATCH_One_FilterConfig)
	e.DELETE("/filter/:filter_name", DELETE_FilterConfig)

	// route to filter association
	e.POST("/service/:service_name/route/:route_name/filter/:filter_name",
		POST_Service_Route_FilterConfig_Association)
	e.GET("/service/:service_name/route/:route_name/filter/:filter_name",
		GET_Service_Route_FilterConfig_Association)
	e.GET("/service/:service_name/route/:route_name/filter",
		GET_Service_Route_All_FilterConfig_Association)
	e.DELETE("/service/:service_name/route/:route_name/filter/:filter_name",
		DELETE_Service_Route_FilterConfig_Association)

	// service to filter association
	e.POST("/service/:service_name/filter/:filter_name",
		POST_Service_FilterConfig_Association)
	e.GET("/service/:service_name/filter/:filter_name",
		GET_Service_FilterConfig_Association)
	e.GET("/service/:service_name/filter",
		GET_Service_All_FilterConfig_Association)
	e.DELETE("/service/:service_name/filter/:filter_name",
		DELETE_Service_FilterConfig_Association)
}
