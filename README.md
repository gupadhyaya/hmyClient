# Steps
* Generate contract go file using go-ethereum abigen cmdline tool
```
abigen --bin Store_sol_Store.bin --abi Store_sol_Store.abi --pkg store --out Store.go
```
* Change Store.go to point to harmony types
```
    ethereum "github.com/ethereum/go-ethereum"
	ethAbi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/harmony-one/harmony/accounts/abi"
	"github.com/harmony-one/harmony/accounts/abi/bind"
	"github.com/harmony-one/harmony/core/types"
```
* Use a custom branch of harmony instead of main repo, pull it and point it in the go.mod replace field
```
https://github.com/gupadhyaya/harmony/tree/abigen
```
* Deploy contract using `go run deploy.go`, make sure to add your deployer key in the code