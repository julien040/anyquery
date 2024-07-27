package main

import (
	"github.com/julien040/anyquery/rpc"
)

func main() {
	plugin := rpc.NewPlugin(myRepoCreator, repoListByUserCreator, commitsCreator, issuesCreator, pullRequestsCreator,
		releaseCreator, branchesCreator, contributorsCreator, tagsCreator, followersCreator, myFollowerCreator,
		followingCreator, my_followingCreator, starsCreator, my_starsCreator, gistsCreator, my_gistsCreator, comments_from_issueCreator,
		my_issuesCreator)
	plugin.Serve()
}
