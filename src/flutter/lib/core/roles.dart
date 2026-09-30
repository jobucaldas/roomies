enum HouseRole { admin, member, monitor }

HouseRole? parseRole(String value) {
  switch (value) {
    case 'admin':
      return HouseRole.admin;
    case 'member':
      return HouseRole.member;
    case 'monitor':
      return HouseRole.monitor;
    default:
      return null;
  }
}

bool canManage(HouseRole role) => role == HouseRole.admin;

bool canCreateExpenseOrNote(HouseRole role) => role != HouseRole.monitor;

bool canMutateOwned(HouseRole role, bool owner) =>
    owner && role != HouseRole.monitor;

bool canMutateNote(HouseRole role, bool author) =>
    author || role == HouseRole.admin;

bool canChangeExpenseVisibility(HouseRole role, bool payer) =>
    payer && role != HouseRole.monitor;
