# Security policy

## Supported versions

Only the latest release receives fixes, and the online demo always runs the
`main` branch.

## What can go wrong

The program reads local files and writes the ones it is asked to; it uses
no network and no server. The online demo is a static page: the program
runs in the browser, and sends nothing anywhere. The problems worth
reporting are therefore, for example:

- a problem or plan file that makes the program crash, hang or use memory
  without bound, or write somewhere other than the files it was given;
- a file whose content runs code in the page of the demo (names are
  displayed there);
- a vulnerable dependency that the weekly `osv-scanner` run has not caught.

## Report a vulnerability

Please do not open a public issue. Use instead
[the private report of GitHub](https://github.com/FrederickFrance/igiveyouarendezvous/security/advisories/new)
("Report a vulnerability", in the Security tab). Without a GitHub account,
write to the maintainer on
[LinkedIn](https://www.linkedin.com/in/frederic-sauvage-rueangpanichpiban)
to agree on a way to send the details.

This is a personal project, maintained on free time: every report is read
and answered, as fast as possible, and a fix is published as a new release,
crediting the reporter if they wish.
