## Embedded Project Example

This example showcases how you can use refresh as an embedded library to reload a project. 
See the ignore config in main.go and make your way through the nested project folders making changes to see how refresh handles it

Keep logs to debug to see refresh monitor all files being changed but choose to ignore based on the configured ruleset
Debug logs will show all rules being checked

Run `go run main.go` to run via the embedded structs setup in main.go file

Run `refresh -f example.toml` to run via CLI and the provided toml file

