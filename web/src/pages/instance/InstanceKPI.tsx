import { Card, Flex, Text } from "@radix-ui/themes";
import {
  Cpu,
  MemoryStick,
  HardDrive,
  Network,
  Link2,
  Activity,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { useLiveData } from "@/contexts/LiveDataContext";
import { useNodeList } from "@/contexts/NodeListContext";
import { formatBytes } from "@/utils/unitHelper";
import type { ReactNode } from "react";

type KpiColor = "green" | "blue" | "orange" | "red" | "violet" | "gray";

const colorBg: Record<KpiColor, string> = {
  green: "var(--green-3)",
  blue: "var(--blue-3)",
  orange: "var(--orange-3)",
  red: "var(--red-3)",
  violet: "var(--violet-3)",
  gray: "var(--gray-3)",
};
const colorFg: Record<KpiColor, string> = {
  green: "var(--green-11)",
  blue: "var(--blue-11)",
  orange: "var(--orange-11)",
  red: "var(--red-11)",
  violet: "var(--violet-11)",
  gray: "var(--gray-11)",
};

const usageColor = (pct: number): KpiColor => {
  if (pct >= 90) return "red";
  if (pct >= 70) return "orange";
  return "green";
};

interface KpiTileProps {
  icon: ReactNode;
  label: string;
  value: ReactNode;
  sub?: ReactNode;
  color?: KpiColor;
  progress?: number; // 0-100
}

const KpiTile = ({
  icon,
  label,
  value,
  sub,
  color = "blue",
  progress,
}: KpiTileProps) => {
  return (
    <Card style={{ overflow: "hidden", padding: 0 }}>
      <div style={{ padding: "12px 14px" }}>
        <Flex justify="between" align="center" mb="2">
          <Text size="2" color="gray">
            {label}
          </Text>
          <div
            style={{
              width: 28,
              height: 28,
              borderRadius: 8,
              background: colorBg[color],
              color: colorFg[color],
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            {icon}
          </div>
        </Flex>
        <Text
          size="6"
          weight="bold"
          style={{ display: "block", lineHeight: 1.1 }}
        >
          {value}
        </Text>
        {sub && (
          <Text
            size="1"
            color="gray"
            style={{ display: "block", marginTop: 4 }}
          >
            {sub}
          </Text>
        )}
      </div>
      {typeof progress === "number" && (
        <div
          style={{
            height: 4,
            width: "100%",
            background: "var(--gray-4)",
          }}
        >
          <div
            style={{
              height: "100%",
              width: `${Math.min(100, Math.max(0, progress))}%`,
              background: colorFg[color],
              transition: "width 0.4s ease",
            }}
          />
        </div>
      )}
    </Card>
  );
};

interface InstanceKPIProps {
  uuid: string;
}

const InstanceKPI = ({ uuid }: InstanceKPIProps) => {
  const { t } = useTranslation();
  const { nodeList } = useNodeList();
  const { live_data } = useLiveData();

  const node = nodeList?.find((n) => n.uuid === uuid);
  const live = live_data?.data.data[uuid];

  const cpuUsage = live?.cpu.usage ?? 0;
  const ramPct = node?.mem_total
    ? ((live?.ram.used ?? 0) / node.mem_total) * 100
    : 0;
  const diskPct = node?.disk_total
    ? ((live?.disk.used ?? 0) / node.disk_total) * 100
    : 0;

  const netUp = live?.network.up ?? 0;
  const netDown = live?.network.down ?? 0;
  const tcp = live?.connections.tcp ?? 0;
  const udp = live?.connections.udp ?? 0;
  const load1 = live?.load.load1 ?? 0;
  const load5 = live?.load.load5 ?? 0;
  const load15 = live?.load.load15 ?? 0;

  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fit, minmax(160px, 1fr))",
        gap: 12,
        width: "100%",
      }}
    >
      <KpiTile
        icon={<Cpu size={16} />}
        label={t("nodeCard.cpu", { defaultValue: "CPU" })}
        value={`${cpuUsage.toFixed(1)}%`}
        sub={
          node?.cpu_cores
            ? `${node.cpu_cores} ${t("instance.kpi.cores", { defaultValue: "核" })}`
            : undefined
        }
        color={usageColor(cpuUsage)}
        progress={cpuUsage}
      />
      <KpiTile
        icon={<MemoryStick size={16} />}
        label={t("nodeCard.ram", { defaultValue: "内存" })}
        value={`${ramPct.toFixed(1)}%`}
        sub={`${formatBytes(live?.ram.used ?? 0)} / ${formatBytes(node?.mem_total ?? 0)}`}
        color={usageColor(ramPct)}
        progress={ramPct}
      />
      <KpiTile
        icon={<HardDrive size={16} />}
        label={t("nodeCard.disk", { defaultValue: "磁盘" })}
        value={`${diskPct.toFixed(1)}%`}
        sub={`${formatBytes(live?.disk.used ?? 0)} / ${formatBytes(node?.disk_total ?? 0)}`}
        color={usageColor(diskPct)}
        progress={diskPct}
      />
      <KpiTile
        icon={<Network size={16} />}
        label={t("nodeCard.networkSpeed", { defaultValue: "网速" })}
        value={
          <span style={{ fontSize: "1rem" }}>
            ↑ {formatBytes(netUp)}/s
            <br />↓ {formatBytes(netDown)}/s
          </span>
        }
        sub={
          live
            ? `${t("instance.kpi.total", { defaultValue: "总" })} ↑ ${formatBytes(live.network.totalUp)} ↓ ${formatBytes(live.network.totalDown)}`
            : undefined
        }
        color="violet"
      />
      <KpiTile
        icon={<Link2 size={16} />}
        label={t("instance.kpi.connections", { defaultValue: "连接数" })}
        value={`${tcp + udp}`}
        sub={`TCP ${tcp} / UDP ${udp}`}
        color="blue"
      />
      <KpiTile
        icon={<Activity size={16} />}
        label={t("instance.kpi.load", { defaultValue: "负载" })}
        value={load1.toFixed(2)}
        sub={`5m ${load5.toFixed(2)} · 15m ${load15.toFixed(2)}`}
        color={
          node?.cpu_cores && load1 > node.cpu_cores
            ? "red"
            : node?.cpu_cores && load1 > node.cpu_cores * 0.7
              ? "orange"
              : "green"
        }
      />
    </div>
  );
};

export default InstanceKPI;
