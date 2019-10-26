set(target_name		enroute-cp)
set(target_image  	"${image_registry}/${target_name}")
set(target_name_push "${target_name}_push")

file(GLOB_RECURSE enroute_project_source ${CMAKE_SOURCE_DIR}/${target_name}/*.go)

# Build container
add_custom_target(${target_name}	ALL DEPENDS ${enroute_project_source})
add_custom_command(TARGET ${target_name}
		  COMMAND GO111MODULE=on GOPATH=${CMAKE_BINARY_DIR}/packaging/ make -C ${CMAKE_SOURCE_DIR}/${target_name} install
		  COMMAND cp ${CMAKE_BINARY_DIR}/packaging/bin/enroute-cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp
          COMMAND cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/Dockerfile.cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/Dockerfile
		  COMMAND docker build ${CMAKE_SOURCE_DIR}/packaging/enroute-cp -t ${target_image}:latest
		  COMMAND find ${CMAKE_SOURCE_DIR}/packaging/${target_name} -name enroute-cp -type f -exec rm {} '\;'
)

#Push container
add_custom_target(${target_name_push}
		DEPENDS ${target_name}
		COMMAND docker push ${image_registry}/enroute-cp:latest
	)

set(target_name_gw		enroute-gw)
set(target_image_gw  	"${image_registry}/${target_name_gw}")
set(target_name_gw_push "${target_name_gw}_push")

file(GLOB_RECURSE enroute_project_source ${CMAKE_SOURCE_DIR}/${target_name}/*.go)

# Build container
add_custom_target(${target_name_gw}	ALL DEPENDS ${enroute_project_source})
add_custom_command(TARGET ${target_name_gw}
		  COMMAND GO111MODULE=on GOPATH=${CMAKE_BINARY_DIR}/packaging/ make -C ${CMAKE_SOURCE_DIR}/${target_name} install
		  COMMAND cp ${CMAKE_BINARY_DIR}/packaging/bin/enroute-cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp
		  COMMAND cp ${CMAKE_BINARY_DIR}/packaging/bin/enroute ${CMAKE_SOURCE_DIR}/packaging/enroute-cp
          COMMAND cp ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/Dockerfile.gw ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/Dockerfile
		  COMMAND docker build ${CMAKE_SOURCE_DIR}/packaging/enroute-cp -t ${target_image_gw}:latest
          # build done, cleanup
		  COMMAND find ${CMAKE_SOURCE_DIR}/packaging/${target_name} -name enroute-cp -type f -exec rm {} '\;'
		  COMMAND find ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/ -name enroute -type f -exec rm {} '\;'
          COMMAND find ${CMAKE_SOURCE_DIR}/packaging/enroute-cp/ -name Dockerfile -type f -exec rm {} '\;'
)

#Push container
add_custom_target(${target_name_push_gw}
		DEPENDS ${target_name_gw}
		COMMAND docker push ${image_registry}/enroute-cp:latest
	)
