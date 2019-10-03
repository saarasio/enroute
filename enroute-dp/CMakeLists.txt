set(target_name	enroute-dp)
file(GLOB_RECURSE enroute_project_source ${CMAKE_CURRENT_DIR}/*.go)
add_custom_target(${target_name}	ALL DEPENDS ${enroute_project_source})
add_custom_command(TARGET ${target_name}
		  COMMAND GO111MODULE=on GOPATH=${CMAKE_SOURCE_DIR}/packaging/staging/ make -C ${CMAKE_SOURCE_DIR}/${target_name} install
)
