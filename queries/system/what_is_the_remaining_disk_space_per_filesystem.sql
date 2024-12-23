/*
title = "What is the remaining disk space per filesystem?"
description = "Get the remaining disk space for each filesystem on the system"
plugins = ["system"]
author = "julien040"
tags = ["system", "disk", "filesystem", "usage"]
*/

SELECT
    p.device,
    s.free AS remaining_disk_space
FROM
    system_partitions p
    JOIN system_partition_stats s ON p.mountpoint = s.mountpoint;