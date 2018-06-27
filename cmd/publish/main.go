package main

import "fmt"

func main() {
	fmt.Println("publishing")

	// output directory format
	// dist -|
	//		drafts.json
	//		drafts -|
	//			   one-week-off-grid.json
	//		posts.json
	//		posts -|
	//			   im-taking-a-year-off.json
	//			   archives.json
	//			   archives -|
	//						2018.json
	//					    2018 -|
	//							 february.json
	//							 march.json

	// check for dist folder, create it if it doesn't exist
	// read all md files in current folder (recursive)
	//		separate frontmatter (yaml) & markdown
	//		validate frontmatter contains all required fields
	//		convert markdown into html
	//		if draft is set to true, add to drafts array
	//		otherwise add to published array
	// generate main feed of latest 10 posts
	// generate archive feeds eg 2018/feb 2018/march etc
	// push to B2
	// bust cloudflare cache
}
