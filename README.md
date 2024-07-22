A simple CLI program written in go that returns the dominant colors in an image.


### How to run
> [!WARNING]
> This tutorial assumes you're on linux.
1. Clone the repo
```
git clone https://github.com/Reishimanfr/color-finder
```
2. cd into the newly created directory
```
cd color-finder
```
3. Build the source code
```
go build
```
4. Add execute permissions to binary
```
chmod +x ./color-finder
```

### Available flags
`--path`           - Path to the file to be analyzed by color finder (required)<br>
`--return-amount`  - Amount of dominant colors to be returned (default: 5)<br>
`--scaling`        - How much should the input image be scaled down by? (default: 1/4, smaller = accurate but slower)<br>
`--threads`        - Amount of threads to be used by the program. More = faster but more CPU and RAM usage. (default: 20)<br>
`--debug`          - Should additional data useful for debugging be shown? (default: false)<br>
