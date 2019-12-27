import React from 'react';
import { StyleSheet, Text, View, Dimensions, StatusBar, ScrollView } from 'react-native';
import { Notifications } from 'expo';
import { EventSubscription } from 'fbemitter';
import {HOMES, Home, stream} from './networking';
import debounce from 'lodash.debounce';
import moment from 'moment';

interface Event {
  ACPower: boolean;
  AlarmHasOccured: boolean;
  AlarmSounding: boolean;
  ArmedAway: boolean;
  ArmedHome: boolean;
  BacklightOn: boolean;
  BatteryLow: boolean;
  Beeps: number;
  ChimeEnabled: boolean;
  EntryDelayDisabled: boolean;
  Fire: boolean;
  KeypadMessage: string;
  Mode: string;
  PerimeterOnly: boolean;
  ProgrammingMode: boolean;
  RawData: string;
  Ready: boolean;
  SystemIssue: boolean;
  Time: string;
  UnparsedMessage: string;
  Zone: string;
  ZoneBypassed: boolean;
}

interface AlarmViewProps {
  home: Home;
}

interface AlarmViewState {
  events: Event[];
}

export class AlarmView extends React.Component<AlarmViewProps, AlarmViewState> {
  private listener: EventSubscription;

  constructor(props: AlarmViewProps) {
    super(props);

    this.state = {events: []};
  }

  componentDidMount(): void {
    this.updateEvents();

    this.listener = Notifications.addListener(() => {
      this.updateEvents();
    })
  }

  componentWillUnmount(): void {
    this.listener.remove();
  }

  private async updateEvents(): Promise<void> {
    const es = await stream<Event>(this.props.home.endpoint + "/alarm");
    let pending: Event[] = [];
    const update = debounce(() => {
      this.setState(({events}) => {
        events = pending.concat(events);
        pending = [];
        return {events};
      });
    }, 100);
    while (true) {
      const result = await es.read();
      if (result.done) {
        return;
      }
      pending.unshift(result.value);
      update();
    }
  }

  render(): React.ReactNode {
    const {events} = this.state;
    return (
      <ScrollView>
        <View style={styles.container}>
          <Text>Event count={events.length}</Text>
          {events.map((e) => this.renderEvent(e))}
        </View>
      </ScrollView>
    );
  }

  renderEvent(e: Event): React.ReactNode {
    let time = moment(e.Time);
    const now = moment();
    if (time.isAfter(now)) {
      time = now;
    }
    return (
      <View key={e.Time} style={styles.event}>
        <Text>{time.fromNow()}</Text>
        <Text>{e.KeypadMessage}</Text>
      </View>
    )
  }
}

const styles = StyleSheet.create({
  container: {
    padding: 16,
    flex: 1,
    backgroundColor: '#fff',
    alignItems: 'stretch',
    justifyContent: 'flex-start',
  },

  event: {
    marginTop: 5,
    marginBottom: 5,
    padding: 10,
    backgroundColor: "#eee",
    borderRadius: 16,
  }
})
