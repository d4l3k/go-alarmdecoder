import React from 'react';
import { StyleSheet, Text, View, Dimensions, StatusBar } from 'react-native';
import { registerForPushNotificationsAsync } from './notifications';
import { TabView, SceneMap, Route } from 'react-native-tab-view';
import {TemperatureView} from './TemperatureView';
import {AlarmView} from './AlarmView';
import {HOMES} from './networking';

const initialLayout = { width: Dimensions.get('window').width };

const LazyPlaceholder = ({route}: {route: Route}) => (
  <View style={styles.container}>
    <Text>Loading {route.title}…</Text>
  </View>
);

const _renderLazyPlaceholder = ({ route }: {route: Route}) => <LazyPlaceholder route={route} />;

function Router() {
  const [index, setIndex] = React.useState(0);
  const sceneMap: {
    [key: string]: React.ComponentType<any>;
  } = {};
  const routesConfig: Route[] = [];
  for (const home of HOMES) {
    if (home.alarm) {
      const title = `${home.name}\nAlarm`;
      routesConfig.push({
        key: title,
        title: title,
      });
      sceneMap[title] = AlarmView;
    }
    if (home.thermostat) {
      const title = `${home.name}\nThermostat`;
      routesConfig.push({
        key: title,
        title: title,
      });
      sceneMap[title] = TemperatureView;
    }
  }

  const [routes] = React.useState(routesConfig);
  const renderScene = SceneMap(sceneMap);

  return (
    <TabView
      lazy
      navigationState={{ index, routes }}
      renderLazyPlaceholder={_renderLazyPlaceholder}
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
