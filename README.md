## Jamie Golang Learnings

This is a repository simply for trying out different Go projects to learn a bit of Golang. Projects in here are mostly dummy projects, except for the Profitwell backfill project that allows a Paddle Vendor account to be integrated with Profitwell, even if the Paddle account has already been live.

### Profitwell Backfill
Paddle, a payment processor and merchant of record, has built an integration with Profitwell, a free SaaS metrics platform for measuring key subscription metrics like MRR, ARR, expansion, churn, etc. It is simple to enable in Paddle with the help of Paddle support. However, the current integration only covers subscriptions created from the time the integration is enabled going forward. If you already have subscibers in Paddle, the integration is not retroactive.

This is a Go script that has been built using similar methods that Paddle uses to push subscibers into Profitwell. It leverages the Paddle Subscription APIs to fill in this retroactive gap; IE, it can push any subscriptions into Profitwell created prior to enabling the integration.

First create a config file or a set of constants related to your Paddle and Profitwell environments, IE:
```
/* Fill out your configuration options below, and then you can run the script from the command line */
const (
	PaddleAPIURL     = "https://sandbox-vendors.paddle.com"
	PaddleVendorID   = "7"
	PaddleAuthKey    = "bacdaf1fa8dcacd80bcc9829ed5fefaca409cf6121da4aa423"
	ProfitwellAPIKey = "778AB094466C477A43EC0C5D239B6CEA"
	// Push subscriptions with a signup_date less than or equal to the below

	// This should be the UTC date time stamp that you enabled the Paddle Profitwell integration,
	// as anything from that day forward would be handled by Paddle
	EndDate = "2020-01-20 00:00:00"
	DryRun  = true
)
```