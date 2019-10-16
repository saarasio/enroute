set(target_name	enroute-dp)
set(target_image  "${image_registry}/${target_name}")
set(target_name_push "${target_name}_push")
set(target_name_test "${target_name}_test")

file(GLOB_RECURSE enroute_project_source ${CMAKE_SOURCE_DIR}/${target_name}/*.go)

add_custom_target(${target_name}	ALL DEPENDS ${enroute_project_source})
add_custom_command(TARGET ${target_name}
		  COMMAND GO111MODULE=on GOPATH=${CMAKE_BINARY_DIR}/packaging/ make -C ${CMAKE_SOURCE_DIR}/${target_name} install
		  # Make enroute executable available to docker container
		  COMMAND cp ${CMAKE_BINARY_DIR}/packaging/bin/enroute ${CMAKE_SOURCE_DIR}/${target_name}
		  COMMAND make -C ${CMAKE_SOURCE_DIR}/${target_name} container
		  COMMAND docker build ${CMAKE_SOURCE_DIR}/enroute-dp -t ${target_image}:latest
		  # Remove enroute executable that was copied earlier
		  COMMAND find ${CMAKE_SOURCE_DIR}/${target_name} -name enroute -type f -exec rm {} '\;'
)

#Push container
add_custom_target(${target_name_push}
		  DEPENDS ${target_name}
		  COMMAND docker push ${image_registry}/enroute-dp:latest
		  )

add_custom_target(${target_name_test}
          ALL
		  DEPENDS ${target_name}
		  COMMAND make -C ${CMAKE_SOURCE_DIR}/${target_name} test
		  )
