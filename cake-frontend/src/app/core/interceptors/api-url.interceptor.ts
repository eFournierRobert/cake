import { HttpInterceptorFn } from '@angular/common/http';

const baseUrl = 'http://localhost:8080';

export const apiUrlInterceptor: HttpInterceptorFn = (req, next) => {
  if (!req.url.startsWith('/api/')) {
    return next(req);
  }

  const apiReq = req.clone({
    url: `${baseUrl}${req.url.slice(4)}`,
  });

  return next(apiReq);
}
