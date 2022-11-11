package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/xorcare/pointer"
)

type Ranges struct {
	Prefixes []prefixes `json:"prefixes"`
}

type prefixes struct {
	IpPrefix           string `json:"ip_prefix"`
	Region             string `json:"region"`
	Service            string `json:"service"`
	NetworkBorderGroup string `json:"network_border_group"`
}

type Event struct{}

type Response struct {
	Response string `json:"response"`
}

var client *ec2.EC2

func init() {
	sess, err := session.NewSession()

	if err != nil {
		fmt.Println(err)
	}
	client = ec2.New(sess)
}

func updateSGs() error {
	input := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:sg-updater"),
				Values: []*string{
					aws.String("true"),
				},
			},
		},
	}

	result, err := client.DescribeSecurityGroups(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				return err
			}
		} else {
			return err
		}
	}

	fmt.Printf("Found %v SecurityGroups.\n", len(result.SecurityGroups))

	ips := &Ranges{}

	reader := strings.NewReader("")
	tokenReq, err := http.NewRequest("GET", "https://ip-ranges.amazonaws.com/ip-ranges.json", reader)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(tokenReq)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)

	if err = json.Unmarshal(buf.Bytes(), ips); err != nil {
		return err
	}

	var cfIps = make([]prefixes, 0)

	for _, i := range ips.Prefixes {
		if i.Service == "CLOUDFRONT" {
			cfIps = append(cfIps, i)
		}
	}

	fmt.Printf("Will add %v rules (x2 - for HTTP and HTTPS)\n", len(cfIps))

	reqSgs := (len(cfIps) * 2 / 60) + 1
	if len(result.SecurityGroups) < reqSgs {
		fmt.Printf("Not enough Security Groups.  Found %v rules.  Need to multiply by 2 to cover HTTPS, so we need: %v SGs, but got only: %v\n", len(cfIps), reqSgs, len(result.SecurityGroups))
		return errors.New("not enough Security Group")
	}

	//my SG limit is 60 inbound rules. Going with 30 as the loop will create 2 rules per entry. 1 for http and the other for https.
	maxRulesPerSg := 30
	start, end := 0, maxRulesPerSg

	for _, sg := range result.SecurityGroups {
		fmt.Printf("Working on SG: %v\n", *sg.GroupId)

		if err := removeRules(sg); err != nil {
			return err
		}

		ipPerms := []*ec2.IpPermission{}
		in := &ec2.AuthorizeSecurityGroupIngressInput{GroupId: sg.GroupId, IpPermissions: ipPerms}

		ipranges := []*ec2.IpRange{}

		for _, i := range cfIps[start:end] {
			j := i.IpPrefix

			r := ec2.IpRange{
				CidrIp: &j,
			}

			ipranges = append(ipranges, &r)
		}

		iph := &ec2.IpPermission{
			IpProtocol: aws.String("TCP"),
			FromPort:   pointer.Int64(80),
			ToPort:     pointer.Int64(80),
			IpRanges:   ipranges,
		}

		iphs := &ec2.IpPermission{
			IpProtocol: aws.String("TCP"),
			FromPort:   pointer.Int64(443),
			ToPort:     pointer.Int64(443),
			IpRanges:   ipranges,
		}

		in.IpPermissions = append(in.IpPermissions, iph, iphs)

		_, err := client.AuthorizeSecurityGroupIngress(in)

		if err != nil {
			return err
		}

		fmt.Printf("Updated SG with %d IPs (number of rules = x2).\n", len(ipranges))

		start = end
		end = end + maxRulesPerSg

		if end >= len(cfIps) {
			end = len(cfIps)
		}
	}

	return nil
}

// Remove all the inbound rules for the given security group.
func removeRules(sg *ec2.SecurityGroup) error {
	input := &ec2.RevokeSecurityGroupIngressInput{
		GroupId:       sg.GroupId,
		IpPermissions: sg.IpPermissions,
	}

	out, err := client.RevokeSecurityGroupIngress(input)

	if err != nil {
		return err
	}

	fmt.Printf("Removed all inbound rules: %t\n", *out.Return)
	return nil
}

func HandleRequest(ctx context.Context, event Event) (Response, error) {
	if err := updateSGs(); err != nil {
		return Response{Response: err.Error()}, nil

	}
	return Response{Response: "All Good"}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
