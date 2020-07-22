set(target_name		enroutectl)

add_custom_target(${target_name}	ALL DEPENDS ${enroute_project_source})
set(target_name_test "${target_name}_test")

add_custom_command(TARGET ${target_name}
		  COMMAND GO111MODULE=on GOPATH=${CMAKE_BINARY_DIR}/enroutectl/ make -C ${CMAKE_SOURCE_DIR}/enroutectl
)

add_custom_target(${target_name_test} ALL DEPENDS ${target_name}
    COMMAND cd ${CMAKE_SOURCE_DIR}/${target_name} && go test -count=1 ./...
		  )


