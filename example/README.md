## Embedded Project Example

This example showcases how you can use refresh as an embedded library to reload a project. 
See the ignore config in main.go and make your way through the nested project folders making changes to see how refresh handles it

Keep logs to debug to see refresh monitor all files being changed but choose to ignore based on the configured ruleset

Run `go run main.go -f example.toml` to run via the provided toml file
Run `go run main.go` to run via the embedded structs
