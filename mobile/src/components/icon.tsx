import type { LucideIcon } from 'lucide-react-native';
import BadgeCheckIcon from 'lucide-react-native/icons/badge-check';
import BellIcon from 'lucide-react-native/icons/bell';
import BuildingIcon from 'lucide-react-native/icons/building';
import CameraIcon from 'lucide-react-native/icons/camera';
import CheckIcon from 'lucide-react-native/icons/check';
import ChevronLeftIcon from 'lucide-react-native/icons/chevron-left';
import ChevronRightIcon from 'lucide-react-native/icons/chevron-right';
import CircleUserIcon from 'lucide-react-native/icons/circle-user';
import ClockIcon from 'lucide-react-native/icons/clock';
import CoffeeIcon from 'lucide-react-native/icons/coffee';
import CompassIcon from 'lucide-react-native/icons/compass';
import DatabaseIcon from 'lucide-react-native/icons/database';
import DownloadIcon from 'lucide-react-native/icons/download';
import FileTextIcon from 'lucide-react-native/icons/file-text';
import FlagIcon from 'lucide-react-native/icons/flag';
import GlobeIcon from 'lucide-react-native/icons/globe';
import ImageIcon from 'lucide-react-native/icons/image';
import InfoIcon from 'lucide-react-native/icons/info';
import LanguagesIcon from 'lucide-react-native/icons/languages';
import LockIcon from 'lucide-react-native/icons/lock';
import LogOutIcon from 'lucide-react-native/icons/log-out';
import MailIcon from 'lucide-react-native/icons/mail';
import MapPinIcon from 'lucide-react-native/icons/map-pin';
import MessageSquareTextIcon from 'lucide-react-native/icons/message-square-text';
import NavigationIcon from 'lucide-react-native/icons/navigation';
import PencilIcon from 'lucide-react-native/icons/pencil';
import PhoneIcon from 'lucide-react-native/icons/phone';
import PlusIcon from 'lucide-react-native/icons/plus';
import ReceiptIcon from 'lucide-react-native/icons/receipt';
import RefreshCwIcon from 'lucide-react-native/icons/refresh-cw';
import ScissorsIcon from 'lucide-react-native/icons/scissors';
import SearchIcon from 'lucide-react-native/icons/search';
import Share2Icon from 'lucide-react-native/icons/share-2';
import ShieldCheckIcon from 'lucide-react-native/icons/shield-check';
import ShoppingBagIcon from 'lucide-react-native/icons/shopping-bag';
import SlidersHorizontalIcon from 'lucide-react-native/icons/sliders-horizontal';
import SmartphoneIcon from 'lucide-react-native/icons/smartphone';
import StarIcon from 'lucide-react-native/icons/star';
import StoreIcon from 'lucide-react-native/icons/store';
import ThumbsUpIcon from 'lucide-react-native/icons/thumbs-up';
import TrashIcon from 'lucide-react-native/icons/trash';
import TriangleAlertIcon from 'lucide-react-native/icons/triangle-alert';
import UtensilsCrossedIcon from 'lucide-react-native/icons/utensils-crossed';
import WifiOffIcon from 'lucide-react-native/icons/wifi-off';
import WrenchIcon from 'lucide-react-native/icons/wrench';
import XIcon from 'lucide-react-native/icons/x';

import type { ThemeColor } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';

// The app's icons, from Lucide (https://lucide.dev, ISC licence). Each is
// imported on its own so only these ship, not the whole set; screens use
// the names here, so the set can be swapped in one place.
const ICONS = {
  'compass': CompassIcon,
  'search': SearchIcon,
  'bell': BellIcon,
  'user': CircleUserIcon,
  'chevron-right': ChevronRightIcon,
  'chevron-left': ChevronLeftIcon,
  'x': XIcon,
  'check': CheckIcon,
  'plus': PlusIcon,
  'sliders': SlidersHorizontalIcon,
  'map-pin': MapPinIcon,
  'navigation': NavigationIcon,
  'phone': PhoneIcon,
  'globe': GlobeIcon,
  'share': Share2Icon,
  'clock': ClockIcon,
  'star': StarIcon,
  'verified': BadgeCheckIcon,
  'shield': ShieldCheckIcon,
  'thumbs-up': ThumbsUpIcon,
  'flag': FlagIcon,
  'message': MessageSquareTextIcon,
  'camera': CameraIcon,
  'image': ImageIcon,
  'pencil': PencilIcon,
  'trash': TrashIcon,
  'receipt': ReceiptIcon,
  'file': FileTextIcon,
  'log-out': LogOutIcon,
  'lock': LockIcon,
  'download': DownloadIcon,
  'smartphone': SmartphoneIcon,
  'mail': MailIcon,
  'building': BuildingIcon,
  'languages': LanguagesIcon,
  'database': DatabaseIcon,
  'info': InfoIcon,
  'alert': TriangleAlertIcon,
  'wifi-off': WifiOffIcon,
  'refresh': RefreshCwIcon,
  'utensils': UtensilsCrossedIcon,
  'coffee': CoffeeIcon,
  'scissors': ScissorsIcon,
  'shopping-bag': ShoppingBagIcon,
  'wrench': WrenchIcon,
  'store': StoreIcon,
} satisfies Record<string, LucideIcon>;

export type IconName = keyof typeof ICONS;

export type IconProps = {
  name: IconName;
  size?: number;
  /** A theme colour; defaults to text. */
  color?: ThemeColor;
  /** Filled rather than outlined (e.g. a selected star). */
  filled?: boolean;
  strokeWidth?: number;
};

/**
 * Icons are decorative by default (hidden from screen readers): the control
 * around them carries the label.
 */
export function Icon({ name, size = 20, color = 'text', filled = false, strokeWidth = 2 }: IconProps) {
  const theme = useTheme();
  const Component = ICONS[name];
  return (
    <Component
      size={size}
      color={theme[color]}
      fill={filled ? theme[color] : 'none'}
      strokeWidth={strokeWidth}
      accessible={false}
      importantForAccessibility="no-hide-descendants"
    />
  );
}
