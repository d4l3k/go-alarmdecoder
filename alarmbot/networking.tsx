import React from 'react';
import { StyleSheet, Text, View, Dimensions, StatusBar, ScrollView, ProgressBarAndroid } from 'react-native';
import '@expo/browser-polyfill';
import fetchStream from 'fetch-readablestream';
// <reference path="ndjson.d.ts" />
import ndjsonStream from 'can-ndjson-stream';

interface Home {
  name: string;
  endpoint: string;
  alarm: boolean;
  thermostat: boolean;
};

export async function retry<T>(f: () => Promise<T>, tries: number = 3): Promise<T> {
  for (let i=0;i<tries;i++) {
    try {
      return await f();
    } catch (err) {
      console.warn("try failed", i, f, err);
      if (i === (tries-1)) {
        throw err;
      }
    }
  }
}

export const HOMES: Home[] = [
  {
    name: 'Seattle',
    endpoint: 'http://192.168.0.18:8080',
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

interface NetworkStatusState {
  inflight: number;
}

interface NetworkStatusProps {}

export class NetworkStatus extends React.Component<NetworkStatusProps, NetworkStatusState> {
  static active: NetworkStatus = null;
  public static track(p: Promise<Response>) {
    NetworkStatus.active.addInflight(1);
    p.finally(() => {
      NetworkStatus.active.addInflight(-1);
    });
  }

  constructor(props: NetworkStatusProps) {
    super(props);
    this.state = {inflight: 0};
    NetworkStatus.active = this;
  }

  addInflight(offset: number) {
    console.log('inflight changed', offset, this.state);
    this.setState(({inflight}) => {
      return {inflight: inflight+offset}
    });
  }

  render(): React.ReactNode {
    return (
      <View>
        {this.renderProgress()}
      </View>
    );
  }

  renderProgress(): React.ReactNode | null {
    if (this.state.inflight == 0) {
      return null;
    }
    return <ProgressBarAndroid
      indeterminate
      styleAttr="Horizontal" />;
  }
}

function fetchWrapper(endpoint: string, req: RequestInit): Promise<Response> {
  req.headers = new Headers(req.headers);
  const p = fetchStream(endpoint, req);
  NetworkStatus.track(p);
  return p;
}

export async function post<T, K>(endpoint: string, body: T): Promise<K> {
  const resp = await fetchWrapper(endpoint, {
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
  const resp = await fetchWrapper(endpoint, {
    method: 'GET',
    headers: {
      ...Authorization,
      Accept: 'application/json',
    },
  });
  return resp.json();
}

export async function stream<K>(endpoint: string): Promise<ReadableStreamDefaultReader<K>> {
  const resp = await fetchWrapper(endpoint, {
    method: 'GET',
    headers: {
      ...Authorization,
      Accept: 'application/json',
    },
  });
  return ndjsonStream<K>(resp.body).getReader();
}
