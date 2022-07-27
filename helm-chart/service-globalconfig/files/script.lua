function envoy_on_request(request_handle)
   request_handle:logInfo("Hello World request");
end

function envoy_on_response(response_handle)
   response_handle:logInfo("Hello World response");
   response_handle:headers():add("Lua-Filter-Says", "Hello")
end
