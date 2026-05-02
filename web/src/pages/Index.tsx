import {
  Callout,
  Card,
  Flex,
  Text,
  Popover,
  IconButton,
  Switch,
} from "@radix-ui/themes";
import { useTranslation } from "react-i18next";
import React, { useEffect, Suspense } from "react";
const NodeDisplay = React.lazy(() => import("../components/NodeDisplay"));
import { formatBytes } from "@/utils/unitHelper";
import { useLiveData } from "../contexts/LiveDataContext";
import { useNodeList } from "@/contexts/NodeListContext";
import type { NodeBasicInfo } from "@/contexts/NodeListContext";
import Loading from "@/components/loading";
import { Settings, X } from "lucide-react";
import { useLocalStorage } from "@/hooks/useLocalStorage";

// Intelligent speed formatting function
const formatSpeed = (bytes: number): string => {
  if (bytes === 0) return "0 B/s";
  const units = ["B/s", "KB/s", "MB/s", "GB/s", "TB/s"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const size = bytes / Math.pow(1024, i);

  // Adaptive decimal places
  let decimals = 2;
  if (i >= 3) decimals = 1; // GB and above: 1 decimal
  if (i <= 1) decimals = 0; // B and KB: no decimals
  if (size >= 100) decimals = 0; // 100+ of any unit: no decimals

  return `${size.toFixed(decimals)} ${units[i]}`;
};

const Index = () => {
  const InnerLayout = () => {
    const [t] = useTranslation();
    const { live_data } = useLiveData();
    const [currentTime, setCurrentTime] = React.useState(
      new Date().toLocaleTimeString(),
    );
    //document.title = t("home_title");
    //#region 节点数据
    const { nodeList, isLoading, error, refresh } = useNodeList();

    // 独立的时间更新定时器
    useEffect(() => {
      const timer = setInterval(() => {
        setCurrentTime(new Date().toLocaleTimeString());
      }, 1000);
      return () => clearInterval(timer);
    }, []);

    // Status cards visibility state
    const [statusCardsVisibility, setStatusCardsVisibility] = useLocalStorage(
      "statusCardsVisibility",
      {
        currentTime: true,
        currentOnline: true,
        regionOverview: true,
        trafficOverview: true,
        networkSpeed: true,
        avgLoad: true,
        expiringSoon: true,
        monthlyTraffic: true,
      },
    );

    // Status cards configuration
    const statusCards = [
      {
        key: "currentTime",
        title: t("current_time"),
        getValue: () => currentTime,
        visible: statusCardsVisibility.currentTime,
      },
      {
        key: "currentOnline",
        title: t("current_online"),
        getValue: () =>
          `${live_data?.data?.online.length ?? 0} / ${nodeList?.length ?? 0}`,
        visible: statusCardsVisibility.currentOnline,
      },
      {
        key: "regionOverview",
        title: t("region_overview"),
        getValue: () =>
          nodeList
            ? Object.entries(
                nodeList.reduce(
                  (acc, item) => {
                    if (live_data?.data.online.includes(item.uuid)) {
                      acc[item.region] = (acc[item.region] || 0) + 1;
                    }
                    return acc;
                  },
                  {} as Record<string, number>,
                ),
              ).length
            : 0,
        visible: statusCardsVisibility.regionOverview,
      },
      {
        key: "trafficOverview",
        title: t("traffic_overview"),
        getValue: () => {
          const data = live_data?.data?.data;
          const online = live_data?.data?.online;
          if (!data || !online) return "↑ 0B / ↓ 0B";
          const onlineSet = new Set(online);
          const values = Object.entries(data)
            .filter(([uuid]) => onlineSet.has(uuid))
            .map(([, node]) => node);
          const up = values.reduce(
            (acc, node) => acc + (node.network.totalUp || 0),
            0,
          );
          const down = values.reduce(
            (acc, node) => acc + (node.network.totalDown || 0),
            0,
          );
          return `↑ ${formatBytes(up)} / ↓ ${formatBytes(down)}`;
        },
        visible: statusCardsVisibility.trafficOverview,
      },
      {
        key: "networkSpeed",
        title: t("network_speed"),
        getValue: () => {
          const data = live_data?.data?.data;
          const online = live_data?.data?.online;
          if (!data || !online) return "↑ 0 B/s / ↓ 0 B/s";
          const onlineSet = new Set(online);
          const values = Object.entries(data)
            .filter(([uuid]) => onlineSet.has(uuid))
            .map(([, node]) => node);
          const up = values.reduce(
            (acc, node) => acc + (node.network.up || 0),
            0,
          );
          const down = values.reduce(
            (acc, node) => acc + (node.network.down || 0),
            0,
          );
          return `↑ ${formatSpeed(up)} / ↓ ${formatSpeed(down)}`;
        },
        visible: statusCardsVisibility.networkSpeed,
      },
      {
        key: "avgLoad",
        title: t("avg_load"),
        getValue: () => {
          const onlineSet = new Set(live_data?.data?.online ?? []);
          const data = live_data?.data?.data ?? {};
          const loads = Object.entries(data)
            .filter(([uuid]) => onlineSet.has(uuid))
            .map(([, n]) => n.load?.load1 ?? 0);
          if (!loads.length) return "0.00";
          return (loads.reduce((a, b) => a + b, 0) / loads.length).toFixed(2);
        },
        visible: statusCardsVisibility.avgLoad,
      },
      {
        key: "expiringSoon",
        title: t("expiring_soon"),
        getValue: () => {
          const now = Date.now();
          const oneDay = 86400 * 1000;
          const expiring = (nodeList ?? [])
            .map((n) => {
              if (!n.expired_at) return null;
              const ts = new Date(n.expired_at).getTime();
              if (!ts) return null;
              const days = Math.floor((ts - now) / oneDay);
              return { node: n, days };
            })
            .filter((x): x is { node: NodeBasicInfo; days: number } => Boolean(x));
          const within7 = expiring.filter((x) => x.days >= 0 && x.days < 7).length;
          const within30 = expiring.filter((x) => x.days >= 0 && x.days < 30).length;
          if (within7 === 0 && within30 === 0) {
            return t("expiring_soon_none", { defaultValue: "无" });
          }
          return `${within7} / ${within30}`;
        },
        getPopover: () => {
          const now = Date.now();
          const oneDay = 86400 * 1000;
          const expiring = (nodeList ?? [])
            .map((n) => {
              if (!n.expired_at) return null;
              const ts = new Date(n.expired_at).getTime();
              if (!ts) return null;
              const days = Math.floor((ts - now) / oneDay);
              return { node: n, days };
            })
            .filter((x): x is { node: NodeBasicInfo; days: number } => Boolean(x))
            .filter((x) => x.days < 30)
            .sort((a, b) => a.days - b.days);
          if (!expiring.length) {
            return (
              <Text size="2" color="gray">
                {t("expiring_soon_none_long", { defaultValue: "近 30 天内无到期节点" })}
              </Text>
            );
          }
          return (
            <Flex direction="column" gap="2" style={{ minWidth: 240 }}>
              <Text size="2" weight="bold">
                {t("expiring_soon_legend", {
                  defaultValue: "7 天内 / 30 天内：",
                })}
              </Text>
              <Flex direction="column" gap="1">
                {expiring.map(({ node, days }) => (
                  <Flex
                    key={node.uuid}
                    justify="between"
                    align="center"
                    gap="3"
                  >
                    <Text size="2" truncate style={{ maxWidth: 180 }}>
                      {node.name}
                    </Text>
                    <Text
                      size="2"
                      color={
                        days < 0 ? "red" : days < 7 ? "red" : days < 14 ? "orange" : "gray"
                      }
                    >
                      {days < 0
                        ? t("nodeCard.expired", { defaultValue: "已过期" })
                        : t("nodeCard.expires_in_days", {
                            count: days,
                            defaultValue: `${days} 天`,
                          })}
                    </Text>
                  </Flex>
                ))}
              </Flex>
            </Flex>
          );
        },
        visible: statusCardsVisibility.expiringSoon,
      },
      {
        key: "monthlyTraffic",
        title: t("monthly_traffic"),
        getValue: () => {
          const data = live_data?.data?.data;
          const online = live_data?.data?.online;
          if (!data || !online) return "↑ 0B / ↓ 0B";
          const onlineSet = new Set(online);
          const values = Object.entries(data)
            .filter(([uuid]) => onlineSet.has(uuid))
            .map(([, node]) => node);
          const up = values.reduce(
            (acc, node) => acc + (node.network.monthlyUp || 0),
            0,
          );
          const down = values.reduce(
            (acc, node) => acc + (node.network.monthlyDown || 0),
            0,
          );
          return `↑ ${formatBytes(up)} / ↓ ${formatBytes(down)}`;
        },
        visible: statusCardsVisibility.monthlyTraffic,
      },
    ];

    useEffect(() => {
      const interval = setInterval(() => {
        refresh();
      }, 5000);
      return () => clearInterval(interval);
    }, [nodeList]);

    if (isLoading) {
      return <Loading />;
    }
    if (error) {
      return <div>Error: {error}</div>;
    }

    //#endregion

    return (
      <>
        <Callouts />
        <Card className="summary-card mx-4 md:text-base text-sm relative">
          <div className="absolute top-2 right-2">
            <Popover.Root>
              <Popover.Trigger>
                <IconButton variant="ghost" size="1">
                  <Settings size={16} />
                </IconButton>
              </Popover.Trigger>
              <Popover.Content width="300px">
                <Flex direction="column" gap="3">
                  <Text size="2" weight="bold">
                    {t("status_settings")}
                  </Text>
                  <Flex direction="column" gap="2">
                    {statusCards.map((card) => (
                      <StatusSettingSwitch
                        key={card.key}
                        label={card.title}
                        checked={card.visible}
                        onCheckedChange={(checked) =>
                          setStatusCardsVisibility({
                            ...statusCardsVisibility,
                            [card.key]: checked,
                          })
                        }
                      />
                    ))}
                  </Flex>
                </Flex>
              </Popover.Content>
            </Popover.Root>
          </div>

          {(() => {
            return (
              <div
                className="grid gap-2"
                style={{
                  gridTemplateColumns: `repeat(auto-fit, minmax(230px, 1fr))`,
                  gridAutoRows: "min-content",
                }}
              >
                {statusCards
                  .filter((card) => card.visible)
                  .map((card) => (
                    <TopCard
                      key={card.key}
                      title={card.title}
                      value={card.getValue()}
                      popoverContent={card.getPopover ? card.getPopover() : undefined}
                    />
                  ))}
              </div>
            );
          })()}
        </Card>
        <Suspense fallback={<div style={{ padding: 16 }}>Loading…</div>}>
          <NodeDisplay
            nodes={nodeList ?? []}
            liveData={live_data?.data ?? { online: [], data: {} }}
          />
        </Suspense>
      </>
    );
  };
  return <InnerLayout />;
};

//#region Callouts
const Callouts = () => {
  const [t] = useTranslation();
  const { showCallout } = useLiveData();
  const ishttps = window.location.protocol === "https:";
  const [httpsWarningDismissed, setHttpsWarningDismissed] = useLocalStorage(
    "httpsWarningDismissed",
    false,
  );
  return (
    <Flex direction="column" gap="2" className="m-2">
      <Callout.Root
        m="2"
        hidden={ishttps || httpsWarningDismissed}
        color="red"
      >
        <Callout.Icon>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="24"
            viewBox="0 0 24 24"
          >
            <path
              fill="currentColor"
              d="M10.03 3.659c.856-1.548 3.081-1.548 3.937 0l7.746 14.001c.83 1.5-.255 3.34-1.969 3.34H4.254c-1.715 0-2.8-1.84-1.97-3.34zM12.997 17A.999.999 0 1 0 11 17a.999.999 0 0 0 1.997 0m-.259-7.853a.75.75 0 0 0-1.493.103l.004 4.501l.007.102a.75.75 0 0 0 1.493-.103l-.004-4.502z"
            />
          </svg>
        </Callout.Icon>
        <Flex justify="between" align="center" gap="2" style={{ flex: 1 }}>
          <Callout.Text>
            <Text size="2" weight="medium">
              {t("warn_https")}
            </Text>
          </Callout.Text>
          <IconButton
            variant="ghost"
            size="1"
            color="red"
            onClick={() => setHttpsWarningDismissed(true)}
            aria-label={t("dismiss", { defaultValue: "关闭" })}
          >
            <X size={16} />
          </IconButton>
        </Flex>
      </Callout.Root>
      <Callout.Root m="2" hidden={showCallout} id="callout" color="tomato">
        <Callout.Icon>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="24"
            height="24"
            viewBox="0 0 24 24"
          >
            <path
              fill="currentColor"
              d="M21.707 3.707a1 1 0 0 0-1.414-1.414L18.496 4.09a4.25 4.25 0 0 0-5.251.604l-1.068 1.069a1.75 1.75 0 0 0 0 2.474l3.585 3.586a1.75 1.75 0 0 0 2.475 0l1.068-1.068a4.25 4.25 0 0 0 .605-5.25zm-11 8a1 1 0 0 0-1.414-1.414l-1.47 1.47l-.293-.293a.75.75 0 0 0-1.06 0l-1.775 1.775a4.25 4.25 0 0 0-.605 5.25l-1.797 1.798a1 1 0 1 0 1.414 1.414l1.798-1.797a4.25 4.25 0 0 0 5.25-.605l1.775-1.775a.75.75 0 0 0 0-1.06l-.293-.293l1.47-1.47a1 1 0 0 0-1.414-1.414l-1.47 1.47l-1.586-1.586z"
            />
          </svg>
        </Callout.Icon>
        <Callout.Text>
          <Text size="2" weight="medium">
            {t("warn_websocket")}
          </Text>
        </Callout.Text>
      </Callout.Root>
    </Flex>
  );
};
// #endregion Callouts
export default Index;

type TopCardProps = {
  title: string;
  value: string | number;
  description?: string;
  popoverContent?: React.ReactNode;
};

const TopCard: React.FC<TopCardProps> = React.memo(
  ({ title, value, description, popoverContent }) => {
    const body = (
      <div className="min-w-52 md:max-w-72 w-full">
        <Flex direction="column" gap="1">
          <label className="text-muted-foreground text-sm">{title}</label>
          <label
            className={
              popoverContent
                ? "font-medium -mt-2 text-md cursor-pointer hover:underline"
                : "font-medium -mt-2 text-md"
            }
          >
            {value}
          </label>
          {description && (
            <Text size="2" color="gray">
              {description}
            </Text>
          )}
        </Flex>
      </div>
    );
    if (!popoverContent) return body;
    return (
      <Popover.Root>
        <Popover.Trigger>
          <div role="button">{body}</div>
        </Popover.Trigger>
        <Popover.Content width="320px">{popoverContent}</Popover.Content>
      </Popover.Root>
    );
  },
);

type StatusSettingSwitchProps = {
  label: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
};

const StatusSettingSwitch: React.FC<StatusSettingSwitchProps> = React.memo(
  ({ label, checked, onCheckedChange }) => {
    return (
      <Flex justify="between" align="center">
        <Text size="2">{label}</Text>
        <Switch checked={checked} onCheckedChange={onCheckedChange} />
      </Flex>
    );
  },
);
