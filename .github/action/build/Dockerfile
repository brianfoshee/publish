FROM golang:1.11

LABEL "com.github.actions.name"="Go Builder"
LABEL "com.github.actions.description"="Build this project"
LABEL "com.github.actions.icon"="code"
LABEL "com.github.actions.color"="green"

LABEL "repository"="https://github.com/brianfoshee/publish"
LABEL "homepage"="https://github.com/brianfoshee/publish"
LABEL "maintainer"="Brian Foshee <brian@brianfoshee.com>"

ADD entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
