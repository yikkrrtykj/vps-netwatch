import React from "react";
import {
  Box,
  Card,
  Flex,
  Text,
  Badge,
  Separator,
  IconButton,
} from "@radix-ui/themes";
import type { LiveData, Record } from "../types/LiveData";
import UsageBar from "./UsageBar";
import Flag from "./Flag";
const Sparkline = React.lazy(() => import("./Sparkline"));
import { useTranslation } from "react-i18next";
import Tips from "./ui/tips";
import type { TFunction } from "i18next";
import { Link } from "react-router-dom";
import { useIsMobile } from "@/hooks/use-mobile";
import type { NodeBasicInfo } from "@/contexts/NodeListContext";
import PriceTags from "./PriceTags";
import { TrendingUp } from "lucide-react";
import MiniPingChartFloat from "./MiniPingChartFloat";
import { getOSImage, getOSName } from "@/utils";
import { usePublicInfo } from "@/contexts/PublicInfoContext";
import LatencyBadges from "./LatencyBadges";

import { formatBytes } from "@/utils/unitHelper";

// 到期倒计时：返回剩余天数（< 0 表示已过期）
function expirationDaysLeft(expiredAt: string | undefined | null): number | null {
  if (!expiredAt) return null;
  const ts = new Date(expiredAt).getTime();
  if (!ts) return null;
  return Math.floor((ts - Date.now()) / 86400000);
}

/** 格式化秒*/
export function formatUptime(seconds: number, t: TFunction): string {
  if (!seconds || seconds < 0) return t("nodeCard.time_second", { val: 0 });
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  const parts = [];
  if (d) parts.push(`${d} ${t("nodeCard.time_day")}`);
  if (h) parts.push(`${h} ${t("nodeCard.time_hour")}`);
  if (m) parts.push(`${m} ${t("nodeCard.time_minute")}`);
  if (s || parts.length === 0) parts.push(`${s} ${t("nodeCard.time_second")}`);
  return parts.join(" ");
}

interface NodeProps {
  basic: NodeBasicInfo;
  live: Record | undefined;
  online: boolean;
}

const Node = React.memo(({ basic, live, online }: NodeProps) => {
  const [t] = useTranslation();
  const isMobile = useIsMobile();
  const defaultLive = {
    cpu: { usage: 0 },
    ram: { used: 0 },
    disk: { used: 0 },
    network: {
      up: 0,
      down: 0,
      totalUp: 0,
      totalDown: 0,
      monthlyUp: 0,
      monthlyDown: 0,
    },
    ping: {},
  } as Record;

  const liveData = live || defaultLive;

  const memoryUsagePercent = basic.mem_total
    ? (liveData.ram.used / basic.mem_total) * 100
    : 0;
  const diskUsagePercent = basic.disk_total
    ? (liveData.disk.used / basic.disk_total) * 100
    : 0;

  const uploadSpeed = formatBytes(liveData.network.up);
  const downloadSpeed = formatBytes(liveData.network.down);
  const totalUpload = formatBytes(liveData.network.totalUp);
  const totalDownload = formatBytes(liveData.network.totalDown);
  const monthlyUpload = formatBytes(liveData.network.monthlyUp);
  const monthlyDownload = formatBytes(liveData.network.monthlyDown);
  const daysLeft = expirationDaysLeft(basic.expired_at as unknown as string);

  // 到期倒计时颜色
  const expiringColor: "red" | "orange" | "gray" | null =
    daysLeft === null
      ? null
      : daysLeft < 0
        ? "red"
        : daysLeft < 7
          ? "red"
          : daysLeft < 30
            ? "orange"
            : null;
  const expiringText =
    daysLeft === null
      ? null
      : daysLeft < 0
        ? t("nodeCard.expired", { defaultValue: "已过期" })
        : t("nodeCard.expires_in_days", {
            count: daysLeft,
            defaultValue: `${daysLeft} 天后到期`,
          });

  return (
    <Card
      style={{
        width: "100%",
        margin: "0 auto",
        transition: "all 0.2s ease-in-out",
      }}
      id={basic.uuid}
      className="node-card hover:cursor-pointer hover:shadow-lg hover:bg-accent-2"
    >
      <Flex direction="column" gap="3">
        {/* === Header === */}
        <Flex justify="between" align="center" gap="2">
          <Flex
            justify="start"
            align="center"
            gap="2"
            style={{ flex: 1, minWidth: 0 }}
          >
            <Flag flag={basic.region} />
            <Link
              to={`/instance/${basic.uuid}`}
              style={{ flex: 1, minWidth: 0 }}
            >
              <Flex direction="column" style={{ minWidth: 0 }} gap="1">
                <Flex align="center" gap="2" style={{ minWidth: 0 }}>
                  <Text
                    weight="bold"
                    size={isMobile ? "2" : "3"}
                    truncate
                    style={{ maxWidth: "100%" }}
                  >
                    {basic.name}
                  </Text>
                  {!isMobile && (
                    <img
                      src={getOSImage(basic.os)}
                      alt={basic.os}
                      className="w-4 h-4 opacity-70 shrink-0"
                      title={`${getOSName(basic.os)} / ${basic.arch}`}
                    />
                  )}
                </Flex>
                <PriceTags
                  hidden={false}
                  price={basic.price}
                  billing_cycle={basic.billing_cycle}
                  expired_at={basic.expired_at}
                  currency={basic.currency}
                  traffic_limit={basic.traffic_limit}
                  traffic_limit_type={basic.traffic_limit_type}
                  tags={basic.tags || ""}
                  ip4={basic.ipv4}
                  ip6={basic.ipv6}
                />
              </Flex>
            </Link>
          </Flex>
          <Flex gap="2" align="center" style={{ flex: "none" }}>
            {live?.message && <Tips color="#CE282E">{live.message}</Tips>}
            <Badge color={online ? "green" : "red"} variant="soft">
              {online ? t("nodeCard.online") : t("nodeCard.offline")}
            </Badge>
          </Flex>
        </Flex>

        <Separator size="4" />

        {/* === Metrics: CPU / RAM / Disk === */}
        <Flex direction="column" gap="2">
          <UsageBar
            label={t("nodeCard.cpu")}
            value={liveData.cpu.usage}
            accessory={
              online ? (
                <React.Suspense
                  fallback={<div style={{ width: 64, height: 16 }} />}
                >
                  <Sparkline
                    uuid={basic.uuid}
                    field="cpu"
                    width={64}
                    height={16}
                  />
                </React.Suspense>
              ) : null
            }
          />
          <Flex direction="column" gap="0">
            <UsageBar
              label={t("nodeCard.ram")}
              value={memoryUsagePercent}
              accessory={
                online ? (
                  <React.Suspense
                    fallback={<div style={{ width: 64, height: 16 }} />}
                  >
                    <Sparkline
                      uuid={basic.uuid}
                      field="ram"
                      width={64}
                      height={16}
                    />
                  </React.Suspense>
                ) : null
              }
            />
            <Text size="1" color="gray" style={{ marginTop: -2 }}>
              {formatBytes(liveData.ram.used)} /{" "}
              {formatBytes(basic.mem_total)}
            </Text>
          </Flex>
          <Flex direction="column" gap="0">
            <UsageBar label={t("nodeCard.disk")} value={diskUsagePercent} />
            <Text size="1" color="gray" style={{ marginTop: -2 }}>
              {formatBytes(liveData.disk.used)} /{" "}
              {formatBytes(basic.disk_total)}
            </Text>
          </Flex>
        </Flex>

        {/* === Network: 网速 / 本月 / 累计 === */}
        <Flex direction="column" gap="1">
          <Flex justify="between" align="center">
            <Text size="2" color="gray">
              {t("nodeCard.networkSpeed")}
            </Text>
            <Text size="2">
              ↑ {uploadSpeed}/s ↓ {downloadSpeed}/s
            </Text>
          </Flex>
          <Flex justify="between" align="center">
            <Text size="2" color="gray">
              {t("monthly_traffic", { defaultValue: "本月流量" })}
            </Text>
            <Text size="2">
              ↑ {monthlyUpload} ↓ {monthlyDownload}
            </Text>
          </Flex>
          {basic.traffic_limit > 0 ? (
            <Flex direction="column" gap="0">
              <UsageBar
                label={t("nodeCard.totalTraffic")}
                value={getTrafficPercentage(
                  liveData.network.totalUp,
                  liveData.network.totalDown,
                  basic.traffic_limit,
                  basic.traffic_limit_type ?? "sum",
                )}
                max={Infinity}
              />
              <Flex justify="between" align="center" style={{ marginTop: -2 }}>
                <Text size="1" color="gray">
                  ↑ {totalUpload} ↓ {totalDownload}
                </Text>
                <Text size="1" color="gray">
                  {basic.traffic_limit_type
                    ? basic.traffic_limit_type.charAt(0).toUpperCase() +
                      basic.traffic_limit_type.slice(1)
                    : "Sum"}
                  ({formatBytes(basic.traffic_limit)})
                </Text>
              </Flex>
            </Flex>
          ) : (
            <Flex justify="between" align="center">
              <Text size="2" color="gray">
                {t("nodeCard.totalTraffic")}
              </Text>
              <Text size="2">
                ↑ {totalUpload} ↓ {totalDownload}
              </Text>
            </Flex>
          )}
        </Flex>

        {/* === Latency: 单行横滚，无论几个目标都不破版 === */}
        <Box style={{ marginTop: -2 }}>
          <LatencyBadges uuid={basic.uuid} />
        </Box>

        <Separator size="4" />

        {/* === Footer: 运行时间 + 到期 + 趋势图 === */}
        <Flex justify="between" align="center" gap="2">
          <Text size="2" color="gray">
            {online ? formatUptime(liveData.uptime, t) : "-"}
          </Text>
          <Flex gap="2" align="center">
            {expiringColor && expiringText && (
              <Badge color={expiringColor} variant="soft" size="1">
                {expiringText}
              </Badge>
            )}
            <MiniPingChartFloat
              uuid={basic.uuid}
              hours={24}
              trigger={
                <IconButton variant="ghost" size="1">
                  <TrendingUp size="14" />
                </IconButton>
              }
            />
          </Flex>
        </Flex>
      </Flex>
    </Card>
  );
});

export default Node;

type NodeGridProps = {
  nodes: NodeBasicInfo[];
  liveData: LiveData;
};

export const NodeGrid = ({ nodes, liveData }: NodeGridProps) => {
  const { publicInfo } = usePublicInfo();
  const offlineServerPosition =
    publicInfo?.theme_settings?.offlineServerPosition; // "First/Keep/Last"
  // 确保liveData是有效的
  const onlineNodes = liveData && liveData.online ? liveData.online : [];

  // 排序节点：先按权重排序，权重大的靠前，再根据用户设置排序
  const sortedNodes = [...nodes].sort((a, b) => {
    const aIsOnline = onlineNodes.includes(a.uuid);
    const bIsOnline = onlineNodes.includes(b.uuid);

    if (offlineServerPosition === "First") {
      if (!aIsOnline && bIsOnline) return -1;
      if (aIsOnline && !bIsOnline) return 1;
    } else if (offlineServerPosition === "Keep") {
      // keep order
    } else {
      if (aIsOnline && !bIsOnline) return -1;
      if (!aIsOnline && bIsOnline) return 1;
    }
    return a.weight - b.weight;
  });

  return (
    <Box
      className="gap-2 md:gap-4"
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fill, minmax(320px, 1fr))",
        padding: "1rem",
        width: "100%",
        boxSizing: "border-box",
      }}
    >
      {sortedNodes.map((node) => {
        const isOnline = onlineNodes.includes(node.uuid);
        const nodeData =
          liveData && liveData.data ? liveData.data[node.uuid] : undefined;

        return (
          <Node
            key={node.uuid}
            basic={node}
            live={nodeData}
            online={isOnline}
          />
        );
      })}
    </Box>
  );
};

function getTrafficPercentage(
  totalUp: number,
  totalDown: number,
  limit: number,
  type: "max" | "min" | "sum" | "up" | "down",
) {
  if (limit === 0) return 0;
  switch (type) {
    case "max":
      return (Math.max(totalUp, totalDown) / limit) * 100;
    case "min":
      return (Math.min(totalUp, totalDown) / limit) * 100;
    case "sum":
      return ((totalUp + totalDown) / limit) * 100;
    case "up":
      return (totalUp / limit) * 100;
    case "down":
      return (totalDown / limit) * 100;
    default:
      return 0;
  }
}
