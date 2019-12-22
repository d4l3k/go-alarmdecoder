import {Notifications} from 'expo';
import * as Permissions from 'expo-permissions';
import Constants from 'expo-constants';

const PUSH_ENDPOINTS = [
  'https://your-server.com/users/push-token',
];

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

  for (const endpoint of PUSH_ENDPOINTS) {
    fetch(endpoint, {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        token: token,
        installationId: Constants.installationId,
        deviceName: Constants.deviceId,
        nativeAppVersion: Constants.nativeAppVersion,
      }),
    });
  }
}
