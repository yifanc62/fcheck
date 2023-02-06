# FCheck

a File Checker depends on SHA1

## Usage

> Each group of examples behave the same

### Provider: Generate a file list json

```
D:\data> fcheck -g
D:\> fcheck -g -d data
D:\data> fcheck -g f.json
D:\data> fcheck -g -o f.json
```

### Consumer: Check local files with a file list json

```
D:\data> fcheck
D:\> fcheck -d data -i data\f.json
D:\data> fcheck f.json
D:\data> fcheck -i f.json -o diff.json
D:\data> fcheck -o diff.json
```

### Provider: Generate diff package with a file diff json

```
D:\data> fcheck -p
D:\> fcheck -p -d data data\diff.json
D:\data> fcheck -p diff.json
D:\data> fcheck -p -i diff.json
D:\data> fcheck -p -o diff-package
D:\data> fcheck -p -o D:\data\diff-package -i diff.json
```

## TODO

- [ ] Add option of generating batch file
- [ ] Add option of outputting the list of matched files

## License

MIT License
