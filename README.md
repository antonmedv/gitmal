<p align="center"><img src="img/gitmal-color-logo.webp" alt="Gitmal" width="330" height="330"></p>

# Gitmal

Gitmal is a static page generator for Git repositories. Gitmal generates static HTML pages with files, commits,
code highlighting, and markdown rendering.

## Installation

```sh
go install github.com/antonmedv/gitmal@latest
```

## Usage

Run gitmal in the repository dir. Gitmal will generate pages in _./output_ directory.

```sh
gitmal .
```

Run gitmal with `--help` flag, go get a list of available options.

```sh
gitmal --help
```

## Screenshots

<p align="center"><img src="img/gitmal-screenshot-code-highlighting.webp" alt="Gitmal Code Highlighting" width="600"></p>
<p align="center"><img src="img/gitmal-screenshot-file-tree.webp" alt="Gitmal File Tree" width="600"></p>
<p align="center"><img src="img/gitmal-screenshot-files.webp" alt="Gitmal Files Page" width="600"></p>

## Examples

Here are a few repos hosted on my website:
- google/zx at [git.medv.io/zx/](https://git.medv.io/zx/)
- my-badges/my-badges at [git.medv.io/my-badges/](https://git.medv.io/my-badges/)

I tried gitmal on kubernetes repository as well. Generation on my MacBook Air M2 takes around 16 minutes,
and the result weights around 20 GB.

## Themes

Gitmal supports different code highlighting themes. You can customize the theme with `--theme` flag.

```sh
gitmal --theme github-dark
```

## License

[MIT](LICENSE)
