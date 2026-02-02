import type { User } from '$lib/types/user.type';
import APIService from './api-service';

class QrLoginService extends APIService {
	initSession = async () => {
		const res = await this.api.post('/qr-login/init');
		return res.data as { token: string; expiresIn: number };
	};

	getStatus = async (token: string) => {
		const res = await this.api.get(`/qr-login/status/${token}`);
		return res.data as { authorized: boolean };
	};

	confirmSession = async (token: string) => {
		await this.api.post(`/qr-login/confirm/${token}`);
	};

	exchangeSession = async (token: string) => {
		const res = await this.api.post(`/qr-login/exchange/${token}`);
		return res.data as User;
	};
}

export default QrLoginService;
