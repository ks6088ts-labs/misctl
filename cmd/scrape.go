/*
Copyright © 2024 ks6088ts

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/playwright-community/playwright-go"
	"github.com/spf13/cobra"
)

func assertErrorToNilf(message string, err error) {
	if err != nil {
		log.Fatalf(message, err)
	}
}

// scrapeCmd represents the scrape command
var scrapeCmd = &cobra.Command{
	Use:   "scrape",
	Short: "Scrape urls",
	Long:  `Scrape urls`,
	Run: func(cmd *cobra.Command, args []string) {
		// Parse flags
		urls, err := cmd.Flags().GetStringArray("url")
		assertErrorToNilf("failed to parse `url`: %w", err)
		dir, err := cmd.Flags().GetString("dir")
		assertErrorToNilf("failed to parse `dir`: %w", err)
		headless, err := cmd.Flags().GetBool("headless")
		assertErrorToNilf("failed to parse `headless`: %w", err)

		// Create output directory
		cwd, err := os.Getwd()
		assertErrorToNilf("could not get cwd: %w", err)
		err = os.MkdirAll(filepath.Join(cwd, dir), os.ModePerm)
		assertErrorToNilf("could not create output directory: %w", err)

		// Scrape via Playwright
		pw, err := playwright.Run()
		assertErrorToNilf("could not launch playwright: %w", err)
		browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(headless),
		})
		assertErrorToNilf("could not launch Chromium: %w", err)
		context, err := browser.NewContext()
		assertErrorToNilf("could not create context: %w", err)
		page, err := context.NewPage()
		assertErrorToNilf("could not create page: %w", err)

		// TODO: parallelize
		for _, url := range urls {
			fmt.Printf("Scraping %s\n", url)
			_, err := page.Goto(url, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			})
			assertErrorToNilf("could not goto: %w", err)

			fileName, err := getFileName(url)
			assertErrorToNilf("could not get file name: %w", err)

			_, err = page.Screenshot(playwright.PageScreenshotOptions{
				Path: playwright.String(filepath.Join(cwd, dir, fileName)),
			})
			assertErrorToNilf("could not take screenshot: %w", err)
		}

		// Close browser
		err = browser.Close()
		assertErrorToNilf("could not close browser: %w", err)
		err = pw.Stop()
		assertErrorToNilf("could not stop playwright: %w", err)
	},
}

func getFileName(url string) (string, error) {
	md5 := md5.New()
	_, err := md5.Write([]byte(url))
	if err != nil {
		return "", fmt.Errorf("could not write md5: %w", err)
	}
	return fmt.Sprintf("%x.png", md5.Sum(nil)), nil
}

func init() {
	rootCmd.AddCommand(scrapeCmd)

	scrapeCmd.Flags().StringArrayP("url", "u", []string{}, "URL to scrape")
	scrapeCmd.Flags().StringP("dir", "d", "artifacts", "Output directory")
	scrapeCmd.Flags().BoolP("headless", "m", true, "Headless mode")

	assertErrorToNilf("could not mark `url` as required: %w", scrapeCmd.MarkFlagRequired("url"))
}
