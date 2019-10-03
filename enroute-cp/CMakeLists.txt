set(target_name		enroute-cp)
set(target_image  	"${image_registry}/${target_name}")
set(target_name_push "${target_name}_push")

file(GLOB_RECURSE enroute_project_source ${CMAKE_SOURCE_DIR}/${target_name}/*.go)

# Build container
add_custom_target(${target_name}	ALL DEPENDS ${enroute_project_source})
add_custom_command(TARGET ${target_name}
		  COMMAND GO111MODULE=on GOPATH=${CMAKE_BINARY_DIR}/packaging/ make -C ${CMAKE_SOURCE_DIR}/${target_name} install
		  COMMAND cp ${CMAKE_BINARY_DIR}/packaging/bin/enroute-cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp
		  COMMAND docker build ${CMAKE_SOURCE_DIR}/packaging/enroute-cp -t ${target_image}:latest
		  COMMAND find ${CMAKE_SOURCE_DIR}/packaging/${target_name} -name enroute-cp -type f -exec rm {} '\;'
)

#Push container
add_custom_target(${target_name_push}
		DEPENDS ${target_name}
		COMMAND docker push ${image_registry}/enroute-cp:latest .
	)
