package main

import (
	"crypto/md5"
	"crypto/sha256"

	"github.com/bwmarrin/discordgo"
)

// Corresponds to the first snoflake possible by the Discord API.
// It corresponds to the date 2015-01-01T00:00:00.000Z with a worker ID of 0 and a sequence number of 0.
//
// It is used to find the first object in a Discord API endpoint.
const firstSnowflake = "0000000000000000000000000000000000000000000000000000000000000000"

// Return the MD5 and sha256 hash of a token
func hashToken(token string) ([]byte, []byte) {
	md5h := md5.Sum([]byte(token))
	sha256h := sha256.Sum256([]byte(token))

	return md5h[:], sha256h[:]
}

var auditLogReasonDeleted = discordgo.WithAuditLogReason("Deleted by Anyquery - A SQL query engine for APIs")
var auditLogReasonUpdated = discordgo.WithAuditLogReason("Updated by Anyquery - A SQL query engine for APIs")
var auditLogReasonCreated = discordgo.WithAuditLogReason("Created by Anyquery - A SQL query engine for APIs")
