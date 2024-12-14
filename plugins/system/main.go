package main

import (
	"github.com/julien040/anyquery/rpc"
)

func main() {
	plugin := rpc.NewPlugin(
		processesCreator,
		process_statusCreator,
		process_memoryCreator,
		process_filesCreator,
		process_networksCreator,
		process_statsCreator,
		cpu_infosCreator,
		cpu_statsCreator,
		swapsCreator,
		memory_statsCreator,
		partitionsCreator,
		partition_usageCreator,
		network_interfacesCreator,
		network_statsCreator,
	)
	plugin.Serve()
}
