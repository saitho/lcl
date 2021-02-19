## Server

### Build binary

```
cd server && go build -o lcl-service .
```

## Website

### Local development:

```
docker run --rm -v `pwd`:/w -w /w -u `id -u`:`id -g` -p 1313:1313 klakegg/hugo:latest-ext server --source=website
```


Building website static files (dev):

```
docker run --rm -v `pwd`:/w -w /w -u `id -u`:`id -g` klakegg/hugo:latest-ext --source=website --destination=../static -b http://localhost:8080
```

Building website static files (prod):

```
docker run --rm -v `pwd`:/w -w /w -u `id -u`:`id -g` klakegg/hugo:latest-ext --source=website --destination=../static
```

## Server + Website

```
docker build . -t saitho/lcl

# Note that the domain lcl.ovh is hardcoded!
docker run --rm -p 8080:8080 saitho/lcl
```
