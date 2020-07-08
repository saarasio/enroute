set(target_name	enroute-dp)
#set(target_tag v0.3.0)
set(target_tag latest)
set(target_image  "${image_registry}/${target_name}:${target_tag}")
set(target_name_push "${target_name}_push")
set(target_name_test "${target_name}_test")
set(target_name_updategenerated "${target_name}_updategenerated")

file(GLOB_RECURSE enroute_project_source ${CMAKE_SOURCE_DIR}/${target_name}/*.go)

# Build enroute-dp
add_custom_target(${target_name}	ALL DEPENDS ${enroute_project_source})
add_custom_command(TARGET ${target_name}
		  COMMAND GO111MODULE=on GOPATH=${CMAKE_BINARY_DIR}/packaging/ make -C ${CMAKE_SOURCE_DIR}/${target_name} install
		  # Make enroute executable available to Dockerfile
		  COMMAND cp ${CMAKE_BINARY_DIR}/packaging/bin/enroute ${CMAKE_SOURCE_DIR}/${target_name}
		  # Make redis-server executable available to Dockerfile
		  COMMAND cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/redis-server ${CMAKE_SOURCE_DIR}/${target_name}
		  COMMAND docker build ${CMAKE_SOURCE_DIR}/enroute-dp -t ${target_image}
		  # Remove enroute executable that was copied earlier
		  COMMAND find ${CMAKE_SOURCE_DIR}/${target_name} -name enroute -type f -exec rm {} '\;'
		  # Remove redis-server executable that was copied earlier
		  COMMAND find ${CMAKE_SOURCE_DIR}/${target_name} -name redis-server -type f -exec rm {} '\;'
)

# Update generated crd code
add_custom_target(${target_name_updategenerated}
    COMMAND make -C ${CMAKE_SOURCE_DIR}/${target_name} updategenerated
    )

# Push container
add_custom_target(${target_name_push} DEPENDS ${target_name}
		  COMMAND docker push ${target_image}
		  )

# Run tests
add_custom_target(${target_name_test} ALL DEPENDS ${target_name}
		  COMMAND make -C ${CMAKE_SOURCE_DIR}/${target_name} test
		  )
