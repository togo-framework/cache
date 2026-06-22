# cache

The `cache` provider plugin for [togo](https://github.com/togo-framework/togo).

## Install

```bash
togo install togo-framework/cache
```

On import it self-registers with the kernel (priority-ordered provider). Access it
via the app container in your handlers/actions (e.g. `a.CACHE`). Swap the default by
registering another provider for the same capability.
