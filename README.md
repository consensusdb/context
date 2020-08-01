# context

Dependency Injection Runtime Framework

All injections happens on runtime and took O(n*m) complexity, where n - number of interfaces, m - number of services.
In golang we need to check each interface with each instance to know if they are compatible. 
All injectable fields must have tag `inject` and be public.

### Usage

SpringFramework-like golang DI framework.

Example:
```

type storageService struct {
    logger *zap.Logger  `inject`
}

type userService struct {
	app.Storage  `inject`
    logger *zap.Logger  `inject`
}

type configService struct {
	app.Storage  `inject`
    logger *zap.Logger  `inject`
}

func Initialize() (context.Context, error) {
    logger, _ := newLogger()
	return context.Create(
		logger,
		storage,
		&userService{},
		&configService{})
}

beans := ctx.Bean("app.UserService")
b, ok := ctx.Bean(reflect.TypeOf((*app.UserService)(nil)).Elem())
```

