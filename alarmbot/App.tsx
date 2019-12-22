import React from 'react';
import { StyleSheet, Text, View, Dimensions, StatusBar } from 'react-native';
import { registerForPushNotificationsAsync } from './notifications';
import { TabView, SceneMap } from 'react-native-tab-view';
import {TemperatureView} from './TemperatureView'

const AlarmView = () => {
  return (
    <View style={styles.container}>
      <Text>Alarm data will appear here!</Text>
    </View>
  );
}

const initialLayout = { width: Dimensions.get('window').width };

function Router() {
  const [index, setIndex] = React.useState(0);
  const [routes] = React.useState([
    { key: 'seattlealarm', title: 'Seattle\nAlarm' },
    { key: 'chaletalarm', title: 'Chalet\nAlarm' },
    { key: 'temperature', title: 'Chalet\nThermostat' },
  ]);

  const renderScene = SceneMap({
    seattlealarm: AlarmView,
    chaletalarm: AlarmView,
    temperature: TemperatureView,
  });

  return (
    <TabView
      navigationState={{ index, routes }}
      renderScene={renderScene}
      onIndexChange={setIndex}
      initialLayout={initialLayout}
      style={styles.router}
    />
  );
}

export default class App extends React.Component<{}, {}> {
  componentDidMount(): void {
    registerForPushNotificationsAsync();
  }

  render() {

    return (
      <Router />
    );
  }
}

const styles = StyleSheet.create({
  router: {
    marginTop: StatusBar.currentHeight,
  },

  container: {
    flex: 1,
    backgroundColor: '#fff',
    alignItems: 'center',
    justifyContent: 'center',
  },
});
