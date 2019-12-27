import React from 'react';
import { StyleSheet, Text, View, Dimensions, StatusBar, ScrollView } from 'react-native';
import { Notifications } from 'expo';
import { EventSubscription } from 'fbemitter';
import {HOMES, get} from './networking';

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

interface AlarmViewState {
  events: Event[];
}

export class AlarmView extends React.Component<{}, AlarmViewState> {
  private listener: EventSubscription;

  constructor(props: any) {
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

  async updateEvents(): Promise<void> {
    const events = await get<Event[]>("http://192.168.1.73:8080/alarm");
    this.setState({events});
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
    return (
      <View key={e.Time} style={styles.event}>
        <Text>{e.Time}</Text>
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
