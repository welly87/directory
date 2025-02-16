import {
  Table,
  TableCaption,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  Tag,
  TagLabel,
  Menu,
  MenuButton,
  MenuItem,
  IconButton,
  MenuList,
  VStack,
  Button
} from '@chakra-ui/react';
import { BsThreeDots } from 'react-icons/bs';
import FormLayout from 'layouts/FormLayout';
import React from 'react';
import { Trans } from '@lingui/react';

type Row = {
  id: string;
  name: string;
  permission: string;
  added: string;
  role: string;
  status: string;
};

const rows = [
  {
    id: '18001',
    name: 'Jones Ferdinand',
    permission: 'Owner',
    added: '14/01/2022',
    role: 'Compliance Officer',
    status: 'active'
  },
  {
    id: '18001',
    name: 'Eason Yang',
    permission: 'Editor',
    added: '14/01/2022',
    role: 'Director of Engineering',
    status: 'active'
  },
  {
    id: '18001',
    name: 'Anusha Aggarwal',
    permission: 'Viewer',
    added: '14/01/2022',
    role: 'General Manager',
    status: 'active'
  }
];

const RowItem: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <Tr
      border="1px solid #23A7E0"
      borderRadius={100}
      css={{
        'td:first-child': {
          border: '1px solid #23A7E0',
          borderRight: 'none',
          borderTopLeftRadius: 100,
          borderBottomLeftRadius: 100
        },
        'td:last-child': {
          border: '1px solid #23A7E0',
          borderLeft: 'none',
          borderTopRightRadius: 100,
          borderBottomRightRadius: 100,
          textAlign: 'center'
        },
        'td:not(:first-child):not(:last-child)': {
          borderTop: '1px solid #23A7E0',
          borderBottom: '1px solid #23A7E0'
        }
      }}>
      {children}
    </Tr>
  );
};

const TableRow: React.FC<{ row: Row }> = ({ row }) => {
  return (
    <>
      <RowItem>
        <>
          <Td>{row.id}</Td>
          <Td>{row.name}</Td>
          <Td>{row.permission}</Td>
          <Td>{row.added}</Td>
          <Td>{row.role}</Td>
          <Td>
            <Tag size="md" borderRadius="full" color="white" background="#60C4CA">
              <TagLabel textTransform="capitalize">{row.status}</TagLabel>
            </Tag>
          </Td>
          <Td paddingY={0}>
            <Menu>
              <MenuButton
                as={IconButton}
                icon={<BsThreeDots />}
                background="transparent"
                _active={{ outline: 'none' }}
                _focus={{ outline: 'none' }}
                borderRadius={50}
              />
              <MenuList>
                <MenuItem>Edit</MenuItem>
                <MenuItem>Change Permissions</MenuItem>
                <MenuItem>Deactivate</MenuItem>
              </MenuList>
            </Menu>
          </Td>
        </>
      </RowItem>
    </>
  );
};

const TableRows: React.FC = () => {
  return (
    <>
      {rows.map((row) => (
        <TableRow key={row.id} row={row} />
      ))}
    </>
  );
};

const CollaboratorsSection: React.FC = () => {
  return (
    <FormLayout overflowX={'scroll'}>
      <Table variant="unstyled" css={{ borderCollapse: 'separate', borderSpacing: '0 9px' }}>
        <TableCaption placement="top" textAlign="start" p={0} m={0} fontSize={20}>
          <Trans id="Organization Collaborators">Organization Collaborators</Trans>
        </TableCaption>
        <Thead>
          <Tr>
            <Th>
              <Trans id="User ID">User ID</Trans>
            </Th>
            <Th>
              <Trans id="Name">Name</Trans>
            </Th>
            <Th>
              <Trans id="Permission">Permission</Trans>
            </Th>
            <Th>
              <Trans id="Added">Added</Trans>
            </Th>
            <Th>
              <Trans id="Role">Role</Trans>
            </Th>
            <Th>
              <Trans id="Status">Status</Trans>
            </Th>
            <Th textAlign="center">
              <Trans id="Action">Action</Trans>
            </Th>
          </Tr>
        </Thead>
        <Tbody>
          <TableRows />
        </Tbody>
      </Table>
      <VStack align="center" w="100%">
        <Button>
          <Trans id="Add Contact">Add Contact</Trans>
        </Button>
      </VStack>
    </FormLayout>
  );
};
export default CollaboratorsSection;
