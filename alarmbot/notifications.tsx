import {Notifications} from 'expo';
import * as Permissions from 'expo-permissions';
import Constants from 'expo-constants';
import {HOMES, post} from './networking';

const PUSH_ENDPOINT = '/register';

export async function registerForPushNotificationsAsync(): Promise<void> {
  const { status: existingStatus } = await Permissions.getAsync(
    Permissions.NOTIFICATIONS
  );
  let finalStatus = existingStatus;

  if (existingStatus !== 'granted') {
    const { status } = await Permissions.askAsync(Permissions.NOTIFICATIONS);
    finalStatus = status;
  }

  if (finalStatus !== 'granted') {
    return;
  }

  let token = await Notifications.getExpoPushTokenAsync();

  for (const home of HOMES) {
    const endpoint = home.endpoint + PUSH_ENDPOINT;
    post(endpoint, {
      Token: token,
      InstallationID: Constants.installationId,
      DeviceName: Constants.deviceId,
      NativeAppVersion: Constants.nativeAppVersion,
    });
  }
}
