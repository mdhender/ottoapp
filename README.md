# ottoweb
Web server for OttoMap

# TailwindUI

## License
This project uses both
[Tailwind CSS](https://tailwindcss.com/)
and
[Tailwind UI](https://tailwindui.com/).

### Tailwind CSS license

```text
Copyright (c) Tailwind Labs, Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### Tailwind UI license
Tailwind UI is a commercial template.
It is not available under any open-source license.
Files in the `assets` and `components` directories that contain TailwindUI styling may not be copied, distributed, or used in any other project without first purchasing a license from Tailwind.

## Setup

```bash
nvm install-latest-npm

# this command doesn't work; caniuse-lite is borked
npx update-browserslist-db@latest

npm install -D tailwindcss

npx tailwindcss init

npm install -D @tailwindcss/forms
```

## Usage

```bash
npx tailwindcss -i assets/css/tailwind-input.css -o assets/css/tailwind.css --watch
```

# Build for DO

```bash
GOOS=linux GOARCH=amd64 go build -o ottoweb.exe
```

# Mac Notes

Creating tar files on the Mac is no fun.

```bash
tar -cz --no-xattrs --no-mac-metadata -f assets.tgz assets
```

