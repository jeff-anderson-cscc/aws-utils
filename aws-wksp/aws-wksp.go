package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/workspaces"
	"os"
	"sort"
)

var awsProfile string
var awsRegion string

func init() {
	const (
		PROFILE   = "profile"
		REGION    = "region"
		FILE_NAME = "file"
	)

	flag.StringVar(&awsProfile, PROFILE, "", "the AWS credentials profile to use")
	flag.StringVar(&awsRegion, REGION, "us-east-1", "the AWS region to use")
}

func main() {

	listBundlesOp := flag.Bool("list-bundles", false, "list workspace bundles")
	listWorkspacesOp := flag.Bool("list-workspaces", false, "list workspaces")
	fileName := flag.String("file", "", "file to read from or write to")

	flag.Parse()

	sessionOptions := session.Options{}
	if len(awsProfile) > 0 {
		sessionOptions.Profile = awsProfile
	}
	sessionOptions.Config.Region = &awsRegion
	sess := session.Must(session.NewSessionWithOptions(sessionOptions))

	svc := workspaces.New(sess)

	var allBundles []*workspaces.WorkspaceBundle
	var bundleMap map[string]string

	if *listBundlesOp {
		allBundles = getAllBundles(*svc)
		if len(*fileName) > 0 {
			writeBundleMap(allBundles, fileName)
		} else {
			bundleMapPrinter(allBundles)
		}
	}
	if *listWorkspacesOp {
		if len(allBundles) == 0 {
			allBundles = getAllBundles(*svc)
		}
		bundleMap = makeBundleMap(allBundles)
		workspacePrinter(getWorkspaces(*svc), bundleMap)
	}
}

func writeBundleMap(bundles []*workspaces.WorkspaceBundle, fileName *string) {
	f, err := os.Create(*fileName)
	checkErr(err)
	defer f.Close()

	fmt.Fprintf(f, "\"bundle_id\",\"bundle_name\"\n")
	for _, v := range bundles {
		fmt.Fprintf(f, "\"%v\",\"%v\"\n", *v.BundleId, *v.Name)
	}
	f.Sync()
}

func makeBundleMap(bundles []*workspaces.WorkspaceBundle) map[string]string {
	var bundleMap = make(map[string]string, len(bundles))
	for _, v := range bundles {
		bundleMap[*v.BundleId] = *v.Name
	}
	return bundleMap
}

func bundleMapPrinter(bundleList []*workspaces.WorkspaceBundle) {
	fmt.Printf("%-15v Description\n", "Bundle ID")
	for _, v := range bundleList {
		fmt.Printf("%v   %v\n", *v.BundleId, *v.Name)
	}
}

func getAllBundles(svc workspaces.WorkSpaces) []*workspaces.WorkspaceBundle {
	bundleList := getBundles("AMAZON", svc)
	bundleList = append(bundleList, getBundles("", svc)...)
	sort.Slice(bundleList, func(i, j int) bool {
		return *bundleList[i].Name < *bundleList[j].Name
	})
	return bundleList
}

func getBundles(bundleOwner string, svc workspaces.WorkSpaces) []*workspaces.WorkspaceBundle {

	bundleList := make([]*workspaces.WorkspaceBundle, 0)

	bundlesInput := new(workspaces.DescribeWorkspaceBundlesInput)
	if len(bundleOwner) > 0 {
		bundlesInput.Owner = &bundleOwner
	}

	for {
		bundleOutput, err := svc.DescribeWorkspaceBundles(bundlesInput)
		checkErr(err)
		bundleList = append(bundleList, bundleOutput.Bundles...)

		if bundleOutput.NextToken != nil {
			bundlesInput.SetNextToken(*bundleOutput.NextToken)
		} else {
			break
		}

	}
	return bundleList
}

func getWorkspaces(svc workspaces.WorkSpaces) []*workspaces.Workspace {
	input := new(workspaces.DescribeWorkspacesInput)
	workspaceList := make([]*workspaces.Workspace, 0)

	for {
		workspaceOutput, err := svc.DescribeWorkspaces(input)
		checkErr(err)
		workspaceList = append(workspaceList, workspaceOutput.Workspaces...)

		if workspaceOutput.NextToken != nil {
			input.SetNextToken(*workspaceOutput.NextToken)
		} else {
			break
		}
	}
	return workspaceList
}

func workspacePrinter(workspaceList []*workspaces.Workspace, bundleMap map[string]string) {
	fmt.Printf("%-15v Description\n", "Bundle ID")
	for _, v := range workspaceList {
		fmt.Printf("%v  %-10v  %v  %v\n", *v.WorkspaceId, *v.State, *v.UserName, bundleMap[*v.BundleId])
	}
}

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}
