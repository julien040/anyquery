/*
title = "How much swap is used by the system?"
description = "Retrieve the total amount of swap space used by the system."

plugins = ["system"]

author = "julien040"

tags = ["system", "swap", "usage", "memory"]

arguments = []
*/

SELECT
    sum(used) AS total_swap_used
FROM
    system_swaps;