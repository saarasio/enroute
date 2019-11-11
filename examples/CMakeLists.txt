set(target_name		examples)

add_custom_target(${target_name}	ALL DEPENDS ${enroute_project_source})
add_custom_command(TARGET ${target_name}
		  COMMAND GO111MODULE=on GOPATH=${CMAKE_BINARY_DIR}/examples/ make -C ${CMAKE_SOURCE_DIR}/examples/helloworld
)
