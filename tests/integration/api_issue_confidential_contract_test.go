// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// contractOperations are the REST operations the confidential-issues change adds or changes.
var contractOperations = []struct {
	method, path, operationID, change string
}{
	{"GET", "/repos/{owner}/{repo}/issues", "issueListIssues", "changed"},
	{"POST", "/repos/{owner}/{repo}/issues", "issueCreateIssue", "changed"},
	{"GET", "/repos/{owner}/{repo}/issues/{index}", "issueGetIssue", "changed"},
	{"PATCH", "/repos/{owner}/{repo}/issues/{index}", "issueEditIssue", "changed"},
	{"GET", "/repos/issues/search", "issueSearchIssues", "changed"},
	{"GET", "/repos/{owner}/{repo}/issues/pinned", "repoListPinnedIssues", "changed"},
	{"GET", "/repos/{owner}/{repo}/issues/{index}/timeline", "issueGetCommentsAndTimeline", "changed"},
	{"GET", "/repos/{owner}/{repo}/issues/{index}/comments", "issueGetComments", "changed"},
	{"GET", "/repos/{owner}/{repo}/issues/comments", "issueGetRepoComments", "changed"},
	{"GET", "/repos/{owner}/{repo}/issues/comments/{id}", "issueGetComment", "changed"},
	{"GET", "/repos/{owner}/{repo}/issues/{index}/dependencies", "issueListIssueDependencies", "changed"},
	{"GET", "/repos/{owner}/{repo}/issues/{index}/blocks", "issueListBlocks", "changed"},
	{"GET", "/repos/{owner}/{repo}/issues/{index}/reactions", "issueGetIssueReactions", "changed"},
	{"GET", "/repos/{owner}/{repo}/milestones", "issueGetMilestonesList", "changed"},
	{"GET", "/repos/{owner}/{repo}/milestones/{id}", "issueGetMilestone", "changed"},
	{"GET", "/notifications", "notifyGetList", "changed"},
	{"GET", "/repos/{owner}/{repo}/notifications", "notifyGetRepoList", "changed"},
	{"GET", "/notifications/new", "notifyNewAvailable", "changed"},
	{"GET", "/repos/{owner}/{repo}", "repoGet", "changed"},
	{"GET", "/repositories/{id}", "repoGetByID", "changed"},
	{"GET", "/repos/search", "repoSearch", "changed"},
	{"GET", "/repos/{owner}/{repo}/forks", "listForks", "changed"},
	{"GET", "/orgs/{org}/repos", "orgListRepos", "changed"},
	{"GET", "/teams/{id}/repos", "orgListTeamRepos", "changed"},
	{"GET", "/teams/{id}/repos/{org}/{repo}", "orgListTeamRepo", "changed"},
	{"GET", "/user/repos", "userCurrentListRepos", "changed"},
	{"GET", "/users/{username}/repos", "userListRepos", "changed"},
	{"GET", "/user/starred", "userCurrentListStarred", "changed"},
	{"GET", "/users/{username}/starred", "userListStarred", "changed"},
	{"GET", "/user/subscriptions", "userCurrentListSubscriptions", "changed"},
	{"GET", "/users/{username}/subscriptions", "userListSubscriptions", "changed"},
	{"POST", "/repos/{owner}/{repo}/transfer/accept", "acceptRepoTransfer", "changed"},
	{"POST", "/repos/{owner}/{repo}/transfer/reject", "rejectRepoTransfer", "changed"},
}

// contractWebOperations are the web routes the confidential-issues change adds or changes.
var contractWebOperations = []struct {
	method, path, operationID string
}{
	{"POST", "/{owner}/{repo}/issues/{index}/confidential", "repoUpdateIssueConfidential"},
	{"POST", "/{owner}/{repo}/issues/new", "repoNewIssuePost"},
	{"GET", "/{owner}/{repo}/issues", "repoIssues"},
	{"GET", "/{owner}/{repo}/issues/{index}", "repoViewIssue"},
	{"GET", "/{owner}/{repo}/issues/{index}/info", "repoGetIssueInfo"},
	{"GET", "/attachments/{uuid}", "repoServeAttachment"},
	{"GET", "/{owner}/{repo}/attachments/{uuid}", "repoServeAttachment"},
}

// concretePath fills the path parameters with values the router accepts
func concretePath(path string) string {
	return strings.NewReplacer(
		"{index}", "1", "{id}", "1", "{uuid}", "00000000-0000-0000-0000-000000000000",
		"{owner}", "user2", "{org}", "org3", "{repo}", "repo1", "{username}", "user2",
	).Replace(path)
}

func TestAPIConfidentialOpenAPIContract(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromFile("../../templates/swagger/v1-openapi3.generated.json")
	require.NoError(t, err)
	require.NoError(t, doc.Validate(t.Context()))

	t.Run("schemas expose confidential", func(t *testing.T) {
		for _, name := range []string{"Issue", "CreateIssueOption", "EditIssueOption"} {
			schemaRef := doc.Components.Schemas[name]
			require.NotNil(t, schemaRef, name)
			property := schemaRef.Value.Properties["confidential"]
			require.NotNil(t, property, name)
			assert.True(t, property.Value.Type.Is("boolean"), name)
		}
	})

	t.Run("operations", func(t *testing.T) {
		assert.Len(t, contractOperations, 33)
		for _, contract := range contractOperations {
			pathItem := doc.Paths.Find(contract.path)
			if !assert.NotNil(t, pathItem, contract.path) {
				continue
			}
			operation := pathItem.GetOperation(contract.method)
			if assert.NotNil(t, operation, "%s %s", contract.method, contract.path) {
				assert.Equal(t, contract.operationID, operation.OperationID, "%s %s", contract.method, contract.path)
			}
		}
	})

	t.Run("web-only confidential route is not REST", func(t *testing.T) {
		for path := range doc.Paths.Map() {
			assert.NotContains(t, path, "/confidential")
		}
	})

	t.Run("operations are served by the router", func(t *testing.T) {
		for _, contract := range contractOperations {
			assert.NotEmpty(t, testWebRoutes.FindRoute(contract.method, "/api/v1"+concretePath(contract.path)), "%s %s", contract.method, contract.path)
		}
		for _, contract := range contractWebOperations {
			assert.NotEmpty(t, testWebRoutes.FindRoute(contract.method, concretePath(contract.path)), "%s %s", contract.method, contract.path)
		}
	})

	t.Run("a route absent from the router is detected", func(t *testing.T) {
		assert.Empty(t, testWebRoutes.FindRoute("GET", "/api/v1/repos/user2/repo1/no-such-confidential-route"))
		assert.Empty(t, testWebRoutes.FindRoute("POST", "/user2/repo1/issues/1/no-such-route"))
		assert.Empty(t, testWebRoutes.FindRoute("DELETE", "/api/v1/repos/search"), "wrong method")
	})
}
