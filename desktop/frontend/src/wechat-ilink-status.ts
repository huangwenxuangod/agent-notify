import type { WechatIlinkStatus } from "./api";

type WireStatus = Partial<WechatIlinkStatus> & {
  logged_in?: boolean;
  bound?: boolean;
  session_expired?: boolean;
  user_id?: string;
  qr_url?: string;
  qr_data_url?: string;
  status?: string;
  last_delivery_at?: string;
  last_delivery_error?: string;
};

export function normalizeWechatIlinkStatus(value: WireStatus): WechatIlinkStatus {
  return {
    LoggedIn: value.LoggedIn ?? value.logged_in ?? false,
    Bound: value.Bound ?? value.bound ?? false,
    ...(value.SessionExpired ?? value.session_expired ? { SessionExpired: value.SessionExpired ?? value.session_expired } : {}),
    ...(value.UserID ?? value.user_id ? { UserID: value.UserID ?? value.user_id } : {}),
    ...(value.QRURL ?? value.qr_url ? { QRURL: value.QRURL ?? value.qr_url } : {}),
    ...(value.QRDataURL ?? value.qr_data_url ? { QRDataURL: value.QRDataURL ?? value.qr_data_url } : {}),
    ...(value.Status ?? value.status ? { Status: value.Status ?? value.status } : {}),
    ...(value.LastDeliveryAt ?? value.last_delivery_at ? { LastDeliveryAt: value.LastDeliveryAt ?? value.last_delivery_at } : {}),
    ...(value.LastDeliveryError ?? value.last_delivery_error ? { LastDeliveryError: value.LastDeliveryError ?? value.last_delivery_error } : {}),
  };
}
