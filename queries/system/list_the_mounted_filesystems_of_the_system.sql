/*
title = "List the mounted filesystems of the system"
description = "Retrieve all the mounted filesystems along with their details"

plugins = ["system"]

author = "julien040"

tags = ["system", "filesystems", "partitions", "storage"]

*/

SELECT * FROM system_partitions;