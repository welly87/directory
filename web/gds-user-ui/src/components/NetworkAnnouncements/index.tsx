import React, { FC } from 'react';
import { Stack, Box, Text, Heading, Flex, VStack } from '@chakra-ui/react';
import AnnouncementCarousel from './caroussels';
import * as Sentry from '@sentry/react';
import { t } from '@lingui/macro';
interface NetworkAnnouncementProps {
  datas?: any;
}
const NetworkAnnouncements = (props: NetworkAnnouncementProps) => {
  return (
    <Flex
      border="1px solid #DFE0EB"
      fontFamily={'Open Sans'}
      maxHeight={190}
      fontSize={'18px'}
      bg={'white'}
      p={5}
      mt={10}>
      <Sentry.ErrorBoundary
        fallback={
          <Text color={'red'} pt={20}>{t`An error has occurred to load annoucements`}</Text>
        }>
        <AnnouncementCarousel announcements={props.datas} />
      </Sentry.ErrorBoundary>
    </Flex>
  );
};

NetworkAnnouncements.defaultProps = {
  datas: [
    {
      title: t`Upcoming TRISA Working Group Call`,
      body: t`Join us on Thursday Apr 28 for the TRISA Working Group.`,
      post_date: '2022-04-20',
      author: 'admin@trisa.io'
    },
    {
      title: t`Routine Maintenance Scheduled`,
      body: t`The GDS will be undergoing routine maintenance on Apr 7.`,
      post_date: t`2022-04-01`,
      author: 'admin@trisa.io'
    },
    {
      title: t`Beware the Ides of March`,
      body: t`I have a bad feeling about tomorrow.`,
      post_date: t`2022-03-14`,
      author: 'julius@caesar.com'
    }
  ]
};

export default NetworkAnnouncements;
