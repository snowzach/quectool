import { Card } from "../components/Card";
import { DataField } from "../components/DataField";
import { Spinner } from "../components/Spinner";
import { useApi } from "../hooks/useApi";
import { useDocumentTitle } from "../hooks/useDocumentTitle";
import { getInfo } from "../api/modem";
import { getSysInfo } from "../api/sysinfo";
import { formatUptime, formatLoads } from "../lib/format";

function formatMem(free: number | undefined, total: number | undefined): string {
  if (!free || !total) return "—";
  const mb = (b: number) => (b / 1024 / 1024).toFixed(0);
  return `${mb(free)} / ${mb(total)} MB`;
}

export function DeviceInfo() {
  useDocumentTitle("Device");
  const info = useApi(getInfo);
  const sys = useApi(getSysInfo);

  if (info.loading || sys.loading) return <Spinner />;
  if (info.error) return <div className="text-red-700">{info.error.message}</div>;

  return (
    <div className="grid md:grid-cols-2 gap-4">
      <Card title="Modem">
        <DataField label="Manufacturer" value={info.data?.manufacturer} />
        <DataField label="Model" value={info.data?.model} />
        <DataField label="Firmware" value={info.data?.firmware} />
        <DataField label="IMEI" value={info.data?.imei} />
        <DataField label="Capabilities" value={(info.data?.capabilities ?? []).join(", ")} />
        {info.data?.data_path && <DataField label="Data path" value={info.data.data_path} />}
        {info.data?.usb_protocol && <DataField label="USB protocol" value={info.data.usb_protocol} />}
      </Card>
      <Card title="System">
        <DataField label="Hostname" value={sys.data?.hostname} />
        <DataField label="App uptime" value={formatUptime(sys.data?.process_uptime)} />
        <DataField label="Host uptime" value={formatUptime(sys.data?.uptime)} />
        <DataField label="Load" value={formatLoads(sys.data?.loads)} />
        <DataField label="Mem free / total" value={formatMem(sys.data?.free_ram, sys.data?.total_ram)} />
      </Card>
    </div>
  );
}
