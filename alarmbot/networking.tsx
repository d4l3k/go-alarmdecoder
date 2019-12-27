interface Home {
  name: string;
  endpoint: string;
  alarm: boolean;
  thermostat: boolean;
};
export const HOMES: Home[] = [
  {
    name: 'Seattle',
    endpoint: 'http://192.168.1.73:8080',
    alarm: true,
    thermostat: false,
  },
  {
    name: 'Chalet',
    endpoint: 'http://192.168.1.73:8080',
    alarm: true,
    thermostat: true,
  },
];

const TOKEN = 'foo';
const Authorization = {
  'Authorization': 'Bearer '+TOKEN,
};

export async function post<T, K>(endpoint: string, body: T): Promise<K> {
  const resp = await fetch(endpoint, {
    method: 'POST',
    headers: {
      ...Authorization,
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  })
  return resp.json();
}

export async function get<K>(endpoint: string): Promise<K> {
  const resp = await fetch(endpoint, {
    method: 'GET',
    headers: {
      ...Authorization,
      Accept: 'application/json',
    },
  });
  return resp.json();
}
