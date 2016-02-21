package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/codegangsta/cli"

	"fmt"
	"math/rand"
	"os"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

var (
	// Config reference for AWS API
	sess *session.Session

	// AWS EC2 client object for establishing command
	svc *ec2.EC2
)

func NewCommand() cli.Command {
	return cli.Command{
		Name:  "aws",
		Usage: "Manage machine on AWS",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "region", EnvVar: "AWS_REGION", Usage: "AWS Region"},
			cli.StringFlag{Name: "key", EnvVar: "AWS_ACCESS_KEY_ID", Usage: "AWS access key"},
			cli.StringFlag{Name: "secret", EnvVar: "AWS_SECRET_ACCESS_KEY", Usage: "AWS secret key"},
			cli.StringFlag{Name: "token", EnvVar: "AWS_SESSION_TOKEN", Usage: "session token for temporary credentials"},
		},
		Before: func(c *cli.Context) error {
			// bootstrap EC2 client with command line args
			cfg := aws.NewConfig()
			if region := c.String("region"); region != "" {
				cfg = cfg.WithRegion(region)
			}
			if id, secret, token := c.String("key"), c.String("secret"), c.String("token"); id != "" && secret != "" {
				cfg = cfg.WithCredentials(credentials.NewStaticCredentials(id, secret, token))
			}
			sess = session.New(cfg)
			svc = ec2.New(sess)
			return nil
		},
		Subcommands: []cli.Command{
			{
				Name:  "sync",
				Usage: "bootstrap cluster environment",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "name", Value: "default", Usage: "Name of the profile"},
					cli.StringFlag{Name: "vpc-id", Value: "default", Usage: "AWS VPC identifier"},
				},
				Action: func(c *cli.Context) {
					var profile = make(AWSProfile)
					defer profile.Load().Dump()
					p := &Profile{Name: c.String("name"), Region: *sess.Config.Region}
					vpcInit(c, &p.VPC)
					amiInit(c, &p.Ami)
					if _, ok := profile[p.Region]; !ok {
						profile[p.Region] = make(RegionProfile)
					}
					profile[p.Region][p.Name] = p
				},
			},
			{
				Name:  "create",
				Usage: "create a new EC2 instance",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "name", Value: "default", Usage: "Name of the profile"},
					cli.StringFlag{Name: "type", Value: "t2.micro", Usage: "EC2 instance type"},
					cli.IntFlag{Name: "count", Value: 1, Usage: "EC2 instances to launch in this request"},
					cli.BoolFlag{Name: "private", Usage: "Launch EC2 instance to internal subnet"},
					cli.StringSliceFlag{Name: "group", Usage: "Network security group for user"},
				},
				Action: func(c *cli.Context) {
					var profile = make(AWSProfile)
					profile.Load()
					region, ok := profile[*sess.Config.Region]
					if !ok {
						fmt.Println("Please run sync in the region of choice")
						os.Exit(1)
					}
					var name = c.String("name")
					p, ok := region[name]
					if !ok {
						fmt.Println("Unable to find VPC profile through name")
						os.Exit(1)
					}
					newEC2Inst(c, p)
				},
			},
		},
	}
}