async function call_anyquery(params, userSettings) {
    const anyqueryID = userSettings.anyquery_id;
    if (!anyqueryID || !anyqueryID.length || anyqueryID.length < 2) {
        throw new Error(
            "The Anyquery ID must be set, and not empty. Currently it is: " +
                anyqueryID
        );
    }

    // Get the params
    const method = params.function;
    const arg1 = params.arg1;
    if (!method || !method.length || method.length < 2) {
        throw new Error("The method must be set by the LLM");
    }

    const endpoint = "https://gpt-actions.anyquery.xyz/";

    let toCall, response;
    switch (method) {
        case "listTables":
            toCall = endpoint + "list-tables";
            response = await fetch(toCall, {
                method: "GET",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: "Bearer " + anyqueryID,
                },
            });

            if (!response.ok) {
                throw new Error(
                    "The request to Anyquery failed with status: " +
                        response.status +
                        " and response: " +
                        (await response.text())
                );
            }

            return await response.text();

        case "describeTable":
            toCall = endpoint + "describe-table";
            response = await fetch(toCall, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: "Bearer " + anyqueryID,
                },
                body: JSON.stringify({
                    table_name: arg1,
                }),
            });

            if (!response.ok) {
                throw new Error(
                    "The request to Anyquery failed with status: " +
                        response.status +
                        " and response: " +
                        (await response.text())
                );
            }

            return await response.text();

        case "executeQuery":
            toCall = endpoint + "execute-query";
            response = await fetch(toCall, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: "Bearer " + anyqueryID,
                },
                body: JSON.stringify({
                    query: arg1,
                }),
            });

            if (!response.ok) {
                throw new Error(
                    "The request to Anyquery failed with status: " +
                        response.status +
                        " and response: " +
                        (await response.text())
                );
            }

            return await response.text();

        default:
            throw new Error("The method is not supported by Anyquery.");
    }
}
