# Configuring Substrate shell completion

Substrate 2022.04 introduced shell completion. It is developed against Bash so any shell which supports Bash completion or includes a compatibility layer with Bash completion should be able to configure Substrate's shell completion.

Substrate 2023.05 introduced Fish shell support.

Shell completion makes using Substrate interactively much more pleasant. We recommend adding the appropriate configuration to your `~/.profile` or equivalent.

## Bash / Z shell

```shell
. <(substrate shell-completion)
```

## Fish

Add the following to `~/.config/fish/config.fish`

```shell
. $(substrate shell-completion | psub)
```
