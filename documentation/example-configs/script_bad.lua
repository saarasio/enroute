function envoy_on_request(request_handle)
   request_handle:logInfo("Hello World request");

function envoy_on_response(response_handle)
   response_handle:logInfo("Hello World response");
end
