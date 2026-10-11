<div align="center">
  <img width="150px" alt="a logo depicting mango" src="mango.png">
  <h1>ManGO Provider</h1>
  <a href="https://pkg.go.dev/github.com/luevano/mangoprovider"><img src="https://pkg.go.dev/badge/github.com/luevano/mangoprovider.svg" alt="Go Reference"></a>
</div>

> **Warning**
>
> The API is not stable and may change at any time.

Providers for [libmangal](https://github.com/luevano/libmangal) written in go, included in [mangal](https://github.com/luevano/mangal).

Most of the code/functionality is based heavily on implementations from [Belphemur/mangal](https://github.com/Belphemur/mangal) (which in turn comes from [metafates/mangal](https://github.com/metafates/mangal)) adapted to work with [libmangal](https://github.com/luevano/libmangal)'s [mangal](https://github.com/luevano/mangal).

## Providers

APIs, through [mangodex](https://github.com/luevano/mangodex) and [mangoplus](https://github.com/luevano/mangoplus):

- MangaDex
- MangaPlus, including subscription chapters with a device secret
- MangaPlus Creators, currently not working (its API blocks bots)

Other sites, see [apis](./apis):

- AsuraScans
- FlameComics
- Mangabat, Mangakakalot and Manganato (mangabox), behind Cloudflare: they need [FlareSolverr](https://github.com/FlareSolverr/FlareSolverr) (`Options.Headless`); Manganato only works with a FlareSolverr running Google Chrome, not Chromium

Scrapers: [Mangapill](https://mangapill.com/), [Weeb Central](https://weebcentral.com/) and [Toonily](https://toonily.com/), see [scrapers](./scrapers).

Every provider but MangaPlus Creators can get a manga by its id without a search (`libmangal.MangaByIDProvider`, or a `mid:<id>` search query). Sites with more than one domain fall back to their mirrors when one is down.

## Usage

```go
options := mango.DefaultOptions()
loaders := append(apis.Loaders(options), scrapers.Loaders(options)...)

loader, _, ok := libmangal.FindLoader(loaders, "mango-mangadex")
if !ok {
	panic("provider not found")
}
client, err := libmangal.NewClient(loader, libmangal.DefaultClientOptions())
if err != nil {
	panic(err)
}
defer client.Close()

mangas, err := client.SearchMangas("dandadan")
```

## Credits

Mango logo: [Mango icons created by max.icons - Flaticon](https://www.flaticon.com/free-icons/mango "mango icons")
