import React, { useEffect, useState } from 'react';
import { setCookie, getCookie, removeCookie } from 'utils/cookies';
import { Heading, Stack, Spinner, Flex, Box } from '@chakra-ui/react';
import Loader from 'components/Loader';
import { useNavigate } from 'react-router-dom';
import { useDispatch, useSelector, RootStateOrAny } from 'react-redux';
import { userSelector, logout } from 'modules/auth/login/user.slice';
const Logout: React.FC = () => {
  const dispatch = useDispatch();
  const [isLoading, setIsLoading] = useState(true);
  const navigate = useNavigate();
  useEffect(() => {
    setTimeout(() => {
      dispatch(logout());
      removeCookie('access_token');
      localStorage.removeItem('trs_stepper');
      localStorage.removeItem('persist:root');
      setIsLoading(false);
      navigate('/');
    }, 2000);
  }, [dispatch, navigate]);

  return (
    <Flex height={'100vh'} alignItems="center" justifyContent={'center'}>
      <Stack textAlign="center" py={20}>
        {isLoading && <Loader text="Logout ... " />}
      </Stack>
    </Flex>
  );
};

export default Logout;
