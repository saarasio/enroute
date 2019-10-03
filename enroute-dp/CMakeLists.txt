set(target_name	enroute-dp)
file(GLOB_RECURSE enroute_project_source ${CMAKE_CURRENT_DIR}/*.go)
add_custom_target(${target_name}	ALL DEPENDS ${enroute_project_source})
add_custom_command(TARGET ${target_name}
		  COMMAND GO111MODULE=on GOPATH=${CMAKE_BINARY_DIR}/packaging/ make -C ${CMAKE_SOURCE_DIR}/${target_name} install
		  # Make enroute executable available to docker container
		  COMMAND cp ${CMAKE_BINARY_DIR}/packaging/bin/enroute ${CMAKE_SOURCE_DIR}/${target_name}
		  COMMAND make -C ${CMAKE_SOURCE_DIR}/${target_name} container
		  # Remove enroute executable that was copied earlier
		  COMMAND find ${CMAKE_SOURCE_DIR}/${target_name} -name enroute -type f -exec rm {} '\;'
)
