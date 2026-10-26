const TOKEN_KEY = 'mw_token'

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body !== undefined && !(init.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) headers.set('Authorization', `Bearer ${token}`)

  let res: Response
  try {
    res = await fetch(`/api${path}`, { ...init, headers })
  } catch {
    throw new Error('无法连接服务器，请确认后端已启动')
  }

  if (res.status === 401 && !path.startsWith('/auth/')) {
    localStorage.removeItem(TOKEN_KEY)
    if (location.pathname !== '/login') location.assign('/login')
  }

  const data = await res.json().catch(() => null)
  if (!res.ok) {
    const message =
      data && typeof data === 'object' && 'error' in data
        ? String((data as { error: string }).error)
        : `请求失败 (${res.status})`
    throw new Error(message)
  }
  return data as T
}

export const api = {
  get<T>(path: string): Promise<T> {
    return request<T>(path)
  },
  post<T>(path: string, body?: unknown): Promise<T> {
    return request<T>(path, {
      method: 'POST',
      body: body instanceof FormData ? body : JSON.stringify(body ?? {}),
    })
  },
  patch<T>(path: string, body?: unknown): Promise<T> {
    return request<T>(path, {
      method: 'PATCH',
      body: body instanceof FormData ? body : JSON.stringify(body ?? {}),
    })
  },
  async upload(file: File): Promise<string> {
    const form = new FormData()
    form.append('file', file)
    const data = await request<{ url: string }>('/upload', { method: 'POST', body: form })
    return data.url
  },
}
