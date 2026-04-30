import { Badge, Flex } from "@radix-ui/themes";
import type React from "react";
import { useTranslation } from "react-i18next";
import { formatBytes } from "@/utils/unitHelper";

const PriceTags = ({
  price = 0,
  billing_cycle = 30,
  currency = "",
  expired_at = "",
  traffic_limit = 0,
  traffic_limit_type = "sum",
  tags = "",
  ip4 = "",
  ip6 = "",
  ...props
}: {
  expired_at?: string | number;
  price?: number;
  billing_cycle?: number;
  currency?: string;
  traffic_limit?: number;
  traffic_limit_type?: "sum" | "max" | "min" | "up" | "down";
  tags?: string;
  ip4?: any;
  ip6?: any;
} & React.ComponentProps<typeof Flex>) => {
  void price;
  void billing_cycle;
  void currency;
  const [t] = useTranslation();
  const expiredDate = new Date(expired_at);
  const hasExpiredAt =
    expired_at !== "" && !Number.isNaN(expiredDate.getTime());
  const diffDays = hasExpiredAt
    ? Math.ceil((expiredDate.getTime() - Date.now()) / (1000 * 60 * 60 * 24))
    : 0;
  const trafficModeLabel =
    traffic_limit_type === "max"
      ? "最大"
      : traffic_limit_type === "min"
        ? "最小"
        : traffic_limit_type === "up"
          ? "上传"
          : traffic_limit_type === "down"
            ? "下载"
            : "总量";

  return (
    <Flex gap="1" {...props} wrap="wrap">
      {ip4 && (
        <Badge size="1" variant="soft" className="text-sm" color="green">
          <label className="flex justify-center items-center gap-1 text-xs">
            <div className="border-2 rounded-4xl border-green-500"></div>
            IPv4
          </label>
        </Badge>
      )}

      {ip6 && (
        <Badge size="1" variant="soft" className="text-sm" color="green">
          <label className="flex justify-center items-center gap-1 text-xs">
            <div className="border-2 rounded-4xl border-green-500"></div>
            IPv6
          </label>
        </Badge>
      )}

      {traffic_limit > 0 && (
        <Badge color="blue" size="1" variant="soft" className="text-sm">
          <label className="text-xs">
            {trafficModeLabel} {formatBytes(traffic_limit)}
          </label>
        </Badge>
      )}

      {hasExpiredAt && (
        <Badge
          color={diffDays <= 7 ? "red" : diffDays <= 15 ? "orange" : "green"}
          size="1"
          variant="soft"
          className="text-sm"
        >
          <label className="text-xs">
            {diffDays <= 0
              ? t("common.expired")
              : diffDays > 36500
                ? t("common.long_term")
                : t("common.expired_in", { days: diffDays })}
          </label>
        </Badge>
      )}

      <CustomTags tags={tags} />
    </Flex>
  );
};

const CustomTags = ({ tags }: { tags?: string }) => {
  if (!tags || tags.trim() === "") {
    return <></>;
  }
  const tagList = tags.split(";").filter((tag) => tag.trim() !== "");
  const colors: Array<
    | "ruby"
    | "gray"
    | "gold"
    | "bronze"
    | "brown"
    | "yellow"
    | "amber"
    | "orange"
    | "tomato"
    | "red"
    | "crimson"
    | "pink"
    | "plum"
    | "purple"
    | "violet"
    | "iris"
    | "indigo"
    | "blue"
    | "cyan"
    | "teal"
    | "jade"
    | "green"
    | "grass"
    | "lime"
    | "mint"
    | "sky"
  > = [
    "ruby",
    "gray",
    "gold",
    "bronze",
    "brown",
    "yellow",
    "amber",
    "orange",
    "tomato",
    "red",
    "crimson",
    "pink",
    "plum",
    "purple",
    "violet",
    "iris",
    "indigo",
    "blue",
    "cyan",
    "teal",
    "jade",
    "green",
    "grass",
    "lime",
    "mint",
    "sky",
  ];

  const parseTagWithColor = (tag: string) => {
    const colorMatch = tag.match(/<(\w+)>$/);
    if (colorMatch) {
      const color = colorMatch[1].toLowerCase();
      const text = tag.replace(/<\w+>$/, "");
      if (colors.includes(color as any)) {
        return { text, color: color as (typeof colors)[number] };
      }
    }
    return { text: tag, color: null };
  };

  return (
    <>
      {tagList.map((tag, index) => {
        const { text, color } = parseTagWithColor(tag);
        const badgeColor = color || colors[index % colors.length];

        return (
          <Badge
            key={index}
            color={badgeColor}
            variant="soft"
            className="text-sm"
          >
            <label className="text-xs">{text}</label>
          </Badge>
        );
      })}
    </>
  );
};

export default PriceTags;
