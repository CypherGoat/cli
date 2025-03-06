/*
Copyright © 2025 CypherGoat <contact@cyphergoat.com>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cyphergoat/cli/api"

	"github.com/AlecAivazis/survey/v2"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var swapCmd = &cobra.Command{
	Use:   "swap",
	Short: "Swap cryptocurrencies",
	Long: `Swap command allows you to perform cryptocurrency swaps between two different coins.

This command uses the CypherGoat API to make the exchange.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Define styled output helpers
		titleStyle := color.New(color.FgCyan, color.Bold).SprintFunc()
		successStyle := color.New(color.FgGreen, color.Bold).SprintFunc()
		errorStyle := color.New(color.FgRed, color.Bold).SprintFunc()
		infoStyle := color.New(color.FgYellow).SprintFunc()

		fmt.Println(titleStyle("CypherGoat Exchange"))
		fmt.Println()

		// Collect input with survey
		answers := struct {
			CoinFrom    string
			NetworkFrom string
			CoinTo      string
			NetworkTo   string
			Amount      string
		}{}

		questions := []*survey.Question{
			{
				Name: "CoinFrom",
				Prompt: &survey.Input{
					Message: "Send coin:",
					Help:    "Enter the ticker symbol (e.g., BTC, ETH, SOL)",
				},
				Validate: survey.Required,
			},
			{
				Name: "NetworkFrom",
				Prompt: &survey.Input{
					Message: "Send coin network (leave empty for default):",
					Help:    "Specify network if the asset exists on multiple chains. Leave empty for mainnet (main chain)",
				},
			},
			{
				Name: "CoinTo",
				Prompt: &survey.Input{
					Message: "Receive coin:",
					Help:    "Enter the ticker symbol (e.g., BTC, ETH, SOL)",
				},
				Validate: survey.Required,
			},
			{
				Name: "NetworkTo",
				Prompt: &survey.Input{
					Message: "Receive network (leave empty for default):",
					Help:    "Specify network if the asset exists on multiple chains. Leave empty for mainnet (main chain)",
				},
			},
			{
				Name: "Amount",
				Prompt: &survey.Input{
					Message: "Amount to swap:",
					Help:    fmt.Sprintf("Amount of %s to exchange", strings.ToUpper(answers.CoinFrom)),
				},
				Validate: func(ans interface{}) error {
					strVal, ok := ans.(string)
					if !ok {
						return fmt.Errorf("invalid input")
					}

					var val float64
					_, err := fmt.Sscanf(strVal, "%f", &val)
					if err != nil || val <= 0 {
						return fmt.Errorf("please enter a valid positive number")
					}
					return nil
				},
			},
		}

		err := survey.Ask(questions, &answers)
		if err != nil {
			fmt.Println(errorStyle("Error:"), err)
			return
		}

		// Convert amount to float64 after collecting all inputs
		var amount float64
		_, err = fmt.Sscanf(answers.Amount, "%f", &amount)
		if err != nil {
			fmt.Println(errorStyle("Invalid amount:"), err)
			return
		}

		// Process the network inputs
		if answers.NetworkFrom == "" {
			answers.NetworkFrom = answers.CoinFrom
		}

		if answers.NetworkTo == "" {
			answers.NetworkTo = answers.CoinTo
		}

		coin1 := strings.ToLower(answers.CoinFrom)
		coin2 := strings.ToLower(answers.CoinTo)
		network1 := strings.ToLower(answers.NetworkFrom)
		network2 := strings.ToLower(answers.NetworkTo)

		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " Fetching Rates from Partnered Exchanges..."
		s.Color("cyan")
		s.Start()

		estimates, err := api.FetchEstimateFromAPI(coin1, coin2, amount, false, network1, network2)
		s.Stop()

		if err != nil {
			fmt.Println(errorStyle("Error fetching rates:"), err)
			return
		}

		if len(estimates) == 0 {
			fmt.Println(errorStyle("No exchanges available for this trading pair"))
			return
		}

		fmt.Println()
		fmt.Println(titleStyle("Available Exchange Options"))

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"#", "Exchange", "You Receive", "Exchange Rate"})
		table.SetBorder(false)
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		)

		for i, est := range estimates {
			table.Append([]string{
				fmt.Sprintf("%d", i+1),
				est.ExchangeName,
				fmt.Sprintf("%.8f %s", est.ReceiveAmount, strings.ToUpper(coin2)),
				fmt.Sprintf("1 %s = %.8f %s", strings.ToUpper(coin1), est.ReceiveAmount/amount, strings.ToUpper(coin2)),
			})
		}
		table.Render()
		fmt.Println()

		var exchangeStr string
		prompt := &survey.Input{
			Message: "Select exchange option (enter number):",
		}

		err = survey.AskOne(prompt, &exchangeStr)
		if err != nil {
			fmt.Println(errorStyle("Error:"), err)
			return
		}

		var selectedExchange int
		_, err = fmt.Sscanf(exchangeStr, "%d", &selectedExchange)
		if err != nil || selectedExchange < 1 || selectedExchange > len(estimates) {
			fmt.Println(errorStyle("Invalid selection:"), "Please select a number between 1 and", len(estimates))
			return
		}

		selected := estimates[selectedExchange-1]

		var address string
		addressPrompt := &survey.Input{
			Message: fmt.Sprintf("Your %s receiving address:", strings.ToUpper(coin2)),
		}

		err = survey.AskOne(addressPrompt, &address, survey.WithValidator(survey.Required))
		if err != nil {
			fmt.Println(errorStyle("Error:"), err)
			return
		}

		// Show spinner while creating trade
		s.Suffix = " Processing transaction..."
		s.Start()

		err, tx := api.CreateTradeFromAPI(coin1, coin2, amount, address, selected.ExchangeName, network1, network2)
		s.Stop()

		if err != nil {
			fmt.Println(errorStyle("Error creating transaction:"), err)
			return
		}

		// Display transaction details
		fmt.Println()
		fmt.Println(successStyle("Transaction initiated successfully"))
		fmt.Println()

		// Create details table
		detailsTable := tablewriter.NewWriter(os.Stdout)
		detailsTable.SetBorder(false)
		detailsTable.SetAlignment(tablewriter.ALIGN_LEFT)
		detailsTable.SetHeaderLine(false)
		detailsTable.SetAutoWrapText(false)
		detailsTable.SetColumnSeparator(" ")

		// Use colored output directly in the cell content
		keyStyle := color.New(color.FgCyan, color.Bold).SprintFunc()

		detailsTable.Append([]string{keyStyle("Amount to Send:"), fmt.Sprintf("%.8f %s", amount, strings.ToUpper(coin1))})
		detailsTable.Append([]string{keyStyle("Estimated Receive:"), fmt.Sprintf("%.8f %s", tx.EstimateAmount, strings.ToUpper(coin2))})
		detailsTable.Append([]string{keyStyle("Transaction ID:"), tx.Id})
		detailsTable.Append([]string{keyStyle("Deposit Address:"), tx.Address})
		detailsTable.Append([]string{keyStyle("Exchange Provider:"), selected.ExchangeName})
		detailsTable.Append([]string{keyStyle("Track on cyphergoat.com:"), "https://cyphergoat.com/transaction/" + tx.CGID})

		// Add tracking link if available
		if tx.Track != "" {
			detailsTable.Append([]string{keyStyle("Transaction Status:"), tx.Track})
		}

		detailsTable.Render()

		fmt.Println()
		fmt.Println(infoStyle("Important: Please send the exact amount to the provided deposit address to complete your transaction."))
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(swapCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// swapCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// swapCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
