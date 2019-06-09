package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/workspaces"
	"log"
	"os"
	"sort"
)

var awsProfile string
var awsRegion string

func init() {
	const (
		PROFILE = "profile"
		REGION  = "region"
	)

	flag.StringVar(&awsProfile, PROFILE, "", "the AWS credentials profile to use")
	flag.StringVar(&awsRegion, REGION, "us-east-1", "the AWS region to use")
}

func main() {

	listBundlesOp := flag.Bool("list-bundles", false, "list workspace bundles")
	listWorkspacesOp := flag.Bool("list-workspaces", false, "list workspaces")
	deleteWorkspacesOp := flag.Bool("delete-workspaces", false, "delete workspaces")
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

	if *listBundlesOp && *listWorkspacesOp && len(*fileName) > 0 {
		fmt.Println("specify one of list-bundles or list-workspaces with the file option")
		flag.Usage()
		os.Exit(-1)
	}

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
		if len(*fileName) > 0 {
			writeWorspaceFile(getWorkspaces(*svc), bundleMap, fileName)
		} else {
			workspacePrinter(getWorkspaces(*svc), bundleMap)
		}
	}

	if *deleteWorkspacesOp && len(*fileName) > 0 {
		deleteWorkspaces(svc, fileName)
	}
}

func deleteWorkspaces(svc *workspaces.WorkSpaces, fileName *string) {
	f, err := os.Open(*fileName)
	checkErr(err)
	defer f.Close()

	r := csv.NewReader(f)

	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	records = records[1:]

	fmt.Println("\nDELETING WORKSPACES:\n")
	for _, v := range records {
		fmt.Printf("%v  %-10v  %v  %v\n", v[0], v[1], v[2], v[3])
	}
	fmt.Print("\nThis action is PERMANENT! Type DELETE (in all capital letters) to confirm: ")
	var answer string
	fmt.Scanln(&answer)
	if answer == "DELETE" {
		deleteWorkspacesOperation(*svc, records)
	} else {
		fmt.Println("Workspaces NOT deleted")
	}

}

func deleteWorkspacesOperation(svc workspaces.WorkSpaces, records [][]string) {
	requests := make([]*workspaces.TerminateRequest, 0, len(records))
	for _, v := range records {
		terminateRequest := new(workspaces.TerminateRequest)
		terminateRequest.WorkspaceId = &v[0]
		requests = append(requests, terminateRequest)
	}
	input := workspaces.TerminateWorkspacesInput{
		TerminateWorkspaceRequests: requests,
	}
	fmt.Println(input)
	output, err := svc.TerminateWorkspaces(&input)
	checkErr(err)
	fmt.Println(output)
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
	fmt.Printf("Workspace ID, State, User Name, Bundle\n")
	for _, v := range workspaceList {
		fmt.Printf("%v  %-10v  %v  %v\n", *v.WorkspaceId, *v.State, *v.UserName, bundleMap[*v.BundleId])
	}
}

func writeWorspaceFile(workspaceList []*workspaces.Workspace, bundleMap map[string]string, fileName *string) {
	f, err := os.Create(*fileName)
	checkErr(err)
	defer f.Close()

	fmt.Fprintf(f, "\"workspace_id\",\"state\",\"user_name\",\"bundle\"\n")
	for _, v := range workspaceList {
		fmt.Fprintf(f, "\"%v\",\"%v\",\"%v\",\"%v\"\n", *v.WorkspaceId, *v.State, *v.UserName, bundleMap[*v.BundleId])
	}
	f.Sync()
}

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}
