set(target_name		enroute-cp)
#set(target_tag v0.3.0)
set(target_tag latest)
set(target_image  	"${image_registry}/${target_name}:${target_tag}")
set(target_name_push "${target_name}_push")
set(target_name_test "${target_name}_test")

file(GLOB_RECURSE enroute_project_source ${CMAKE_SOURCE_DIR}/${target_name}/*.go)

# Build container
add_custom_target(${target_name}	ALL DEPENDS ${enroute_project_source})
add_custom_command(TARGET ${target_name}
		  COMMAND GO111MODULE=on GOPATH=${CMAKE_BINARY_DIR}/packaging/ make -C ${CMAKE_SOURCE_DIR}/${target_name} install
		  COMMAND cp ${CMAKE_BINARY_DIR}/packaging/bin/enroute-cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp
          COMMAND cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/Dockerfile.cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/Dockerfile
		  COMMAND docker build ${CMAKE_SOURCE_DIR}/packaging/enroute-cp -t ${target_image}
		  COMMAND find ${CMAKE_SOURCE_DIR}/packaging/${target_name} -name enroute-cp -type f -exec rm {} '\;'
)

#Push container
add_custom_target(${target_name_push}
		DEPENDS ${target_name}
		COMMAND docker push ${target_image}
	)

set(target_name_gw		enroute-gw)
set(target_name_gwi		enroute-gwi)
set(target_image_gw  	"${image_registry}/${target_name_gw}:${target_tag}")
set(target_image_gwi  	"${image_registry}/${target_name_gwi}:${target_tag}")
set(target_name_push_gw "${target_name_gw}_push")
set(target_name_push_gwi "${target_name_gwi}_push")

file(GLOB_RECURSE enroute_project_source ${CMAKE_SOURCE_DIR}/${target_name}/*.go)

# Build container
add_custom_target(${target_name_gw}	ALL DEPENDS ${enroute_project_source})
add_custom_command(TARGET ${target_name_gw}
		  COMMAND GO111MODULE=on GOPATH=${CMAKE_BINARY_DIR}/packaging/ make -C ${CMAKE_SOURCE_DIR}/${target_name} install
		  COMMAND cp ${CMAKE_BINARY_DIR}/packaging/bin/enroute-cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp
		  COMMAND cp ${CMAKE_BINARY_DIR}/packaging/bin/enroute ${CMAKE_SOURCE_DIR}/packaging/enroute-cp
          COMMAND cp ${CMAKE_BINARY_DIR}/enroutectl/bin/enroutectl ${CMAKE_SOURCE_DIR}/packaging/enroute-cp
          COMMAND cp ${CMAKE_SOURCE_DIR}/enroutectl/examples/petstore2.json  ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/petstore.json
		  # build enroute-gw
          COMMAND cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/Dockerfile.gw ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/Dockerfile
		  COMMAND docker build ${CMAKE_SOURCE_DIR}/packaging/enroute-cp -t ${target_image_gw}

		  # build enroute-gwi
          COMMAND cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/Dockerfile.gwi ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/Dockerfile
		  COMMAND docker build ${CMAKE_SOURCE_DIR}/packaging/enroute-cp -t ${target_image_gwi}

          # build done, cleanup
		  COMMAND find ${CMAKE_SOURCE_DIR}/packaging/${target_name} -name enroute-cp -type f -exec rm {} '\;'
		  COMMAND find ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/ -name enroute -type f -exec rm {} '\;'
          COMMAND find ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/ -name Dockerfile -type f -exec rm {} '\;'
          COMMAND find ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/ -name enroutectl -type f -exec rm {} '\;'
          COMMAND find ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/ -name petstore.json -type f -exec rm {} '\;'
)

add_custom_target(${target_name_test} ALL DEPENDS ${target_name}
    COMMAND cd ${CMAKE_SOURCE_DIR}/${target_name} && go test -mod=readonly ./...
		  )

#Push container
add_custom_target(${target_name_push_gw}
		DEPENDS ${target_name_gw}
		COMMAND docker push ${target_image_gw}
	)

#Push container
add_custom_target(${target_name_push_gwi}
		DEPENDS ${target_name_gw}
		COMMAND docker push ${target_image_gwi}
	)
