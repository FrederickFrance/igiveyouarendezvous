# igiveyouarendezvous

[![CI](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml/badge.svg)](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/FrederickFrance/igiveyouarendezvous.svg)](https://pkg.go.dev/github.com/FrederickFrance/igiveyouarendezvous)

[English](README.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Italiano](README.it.md) · [ไทย](README.th.md) · **tlhIngan Hol** · [Ch'ti](README.pcd.md)

nab: DaHjaj ghom Hoch. Qapla'!

`together: true`: ghom Hoch, wa'logh. [Quj](example/roleplay.yaml)

[DaH yIlo'!](https://frederickfrance.github.io/igiveyouarendezvous/)

## nab wa'DIch: mIw

1. **yItlhap**: [Releases](https://github.com/FrederickFrance/igiveyouarendezvous/releases/latest) yIpoSmoH; zip yItlhap:
   Windows: `igiveyouarendezvous-windows.zip`, Mac (Apple): `igiveyouarendezvous-mac-apple.zip`,
   Mac (Intel): `igiveyouarendezvous-mac-intel.zip`, Linux: `igiveyouarendezvous-linux.zip`.
2. **yIghItlh**: `problem.yaml` yIchoH. pong DaneH yIghItlh. Tab yIlo'Qo'!

```yaml
day: 09:00-17:00
breaks: 12:00-13:30
duration: 1h

recruiters:
  Alice:
  Bob:

candidates:
  David:
    meet: [Alice, Bob]
  Emma:
    meet: [Bob]
```

3. **yIQapmoH**: Windows: `igiveyouarendezvous problem.yaml`; Mac, Linux:
   `./igiveyouarendezvous problem.yaml`.
4. **wa'leS**: `igiveyouarendezvous -rest wa-leS.yaml problem.yaml`.

Qapla'! mIw Hoch: [DIvI' Hol](README.md#your-first-schedule-step-by-step).

## jom

```sh
go install github.com/FrederickFrance/igiveyouarendezvous/cmd/igiveyouarendezvous@latest
```

## yIlo'

```sh
igiveyouarendezvous -rest "wa'leS.yaml" example/problem.yaml
```

Ctrl+C: yImev!

De' Hoch: [DIvI' Hol](README.md), [Français](README.fr.md).

## chut

[0BSD](LICENSE), [WTFPL](WTFPL.txt).
